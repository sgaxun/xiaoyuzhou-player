import AppKit

// A nonactivating panel owns no keyboard or mouse input. All AppKit work runs
// on the main thread; newline-delimited updates arrive through the parent pipe.
final class SubtitleView: NSView {
    var text = ""
    var progress: CGFloat = 0
    var attachedToNotch = false
    let subtitleHeight: CGFloat = 38
    let attributes: [NSAttributedString.Key: Any] = [
        .font: NSFont.systemFont(ofSize: 15, weight: .medium),
        .foregroundColor: NSColor.white
    ]
    override func draw(_ dirtyRect: NSRect) {
        NSColor.black.setFill()
        if attachedToNotch {
            // Extend behind the physical camera housing all the way to the
            // screen edge. Only the bottom corners round off: no capsule seam.
            let radius: CGFloat = 18
            let shape = NSBezierPath()
            shape.move(to: NSPoint(x: 0, y: bounds.maxY))
            shape.line(to: NSPoint(x: bounds.maxX, y: bounds.maxY))
            shape.line(to: NSPoint(x: bounds.maxX, y: radius))
            shape.curve(to: NSPoint(x: bounds.maxX - radius, y: 0),
                        controlPoint1: NSPoint(x: bounds.maxX, y: radius * 0.448),
                        controlPoint2: NSPoint(x: bounds.maxX - radius * 0.448, y: 0))
            shape.line(to: NSPoint(x: radius, y: 0))
            shape.curve(to: NSPoint(x: 0, y: radius),
                        controlPoint1: NSPoint(x: radius * 0.448, y: 0),
                        controlPoint2: NSPoint(x: 0, y: radius * 0.448))
            shape.close()
            shape.fill()
        } else {
            NSBezierPath(roundedRect: bounds, xRadius: 14, yRadius: 14).fill()
        }
        let textArea = NSRect(x: 0, y: 0, width: bounds.width, height: subtitleHeight)
        let viewport = textArea.insetBy(dx: 20, dy: 0)
        NSGraphicsContext.saveGraphicsState()
        NSBezierPath(rect: viewport).addClip()
        let size = (text as NSString).size(withAttributes: attributes)
        let offset = max(0, size.width - viewport.width) * progress
        (text as NSString).draw(at: NSPoint(x: viewport.minX - offset,
                                          y: (subtitleHeight - size.height) / 2),
                                 withAttributes: attributes)
        NSGraphicsContext.restoreGraphicsState()
    }
}

final class SubtitleController {
    let panel: NSPanel
    let view = SubtitleView()
    init() {
        panel = NSPanel(contentRect: .zero, styleMask: [.borderless, .nonactivatingPanel],
                        backing: .buffered, defer: false)
        panel.contentView = view
        panel.isOpaque = false
        panel.backgroundColor = .clear
        panel.hasShadow = false
        panel.ignoresMouseEvents = true
        panel.hidesOnDeactivate = false
        panel.isReleasedWhenClosed = false
        panel.level = .statusBar
        panel.collectionBehavior = [.canJoinAllSpaces, .fullScreenAuxiliary, .ignoresCycle]
        NotificationCenter.default.addObserver(forName: NSApplication.didChangeScreenParametersNotification,
                                               object: nil, queue: .main) { [weak self] _ in
            self?.layout()
        }
    }
    func layout() {
        // Prefer the built-in camera screen, regardless of which monitor hosts
        // the menu bar. Fall back to the main screen for desktops/external-only.
        guard let screen = NSScreen.screens.first(where: { $0.safeAreaInsets.top > 0 })
                ?? NSScreen.main ?? NSScreen.screens.first else { return }
        let inset = screen.safeAreaInsets.top
        view.attachedToNotch = inset > 0
        if inset > 0 {
            // Derive the camera housing width from the two usable top regions.
            // Keep the expanded island compact, with enough room for one line.
            let left = screen.auxiliaryTopLeftArea
            let right = screen.auxiliaryTopRightArea
            let notchWidth = (left != nil && right != nil)
                ? max(0, right!.minX - left!.maxX) : 200
            let width = min(max(340, notchWidth + 96), screen.frame.width - 32)
            let height = inset + view.subtitleHeight
            panel.setFrame(NSRect(x: screen.frame.midX - width / 2,
                                  y: screen.frame.maxY - height,
                                  width: width, height: height), display: true)
        } else {
            let width = min(CGFloat(400), screen.frame.width - 32)
            panel.setFrame(NSRect(x: screen.frame.midX - width / 2,
                                  y: screen.visibleFrame.maxY - view.subtitleHeight,
                                  width: width, height: view.subtitleHeight), display: true)
        }
    }
    func update(_ message: [String: Any]) -> [String: Any] {
        view.text = message["text"] as? String ?? ""
        let progress = message["progress"] as? Double ?? 0
        view.progress = CGFloat(progress.isFinite ? max(0, min(1, progress)) : 0)
        if view.text.isEmpty {
            panel.orderOut(nil)
        } else {
            layout()
            view.needsDisplay = true
            panel.orderFrontRegardless()
            panel.displayIfNeeded()
        }
        return ["visible": panel.isVisible, "textLength": view.text.count,
                "progress": view.progress, "ignoresMouseEvents": panel.ignoresMouseEvents,
                "frame": NSStringFromRect(panel.frame)]
    }
}

let app = NSApplication.shared
app.setActivationPolicy(.accessory)
let controller = SubtitleController()
DispatchQueue.global(qos: .userInteractive).async {
    while let line = readLine() {
        guard let data = line.data(using: .utf8),
              let message = try? JSONSerialization.jsonObject(with: data) as? [String: Any] else { continue }
        DispatchQueue.main.sync {
            let status = controller.update(message)
            if let reply = try? JSONSerialization.data(withJSONObject: status) {
                FileHandle.standardOutput.write(reply)
                FileHandle.standardOutput.write(Data([10]))
            }
        }
    }
    // EOF also handles parent crashes; never leave an orphaned subtitle.
    DispatchQueue.main.async { app.terminate(nil) }
}
app.run()
