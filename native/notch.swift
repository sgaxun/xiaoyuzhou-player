import AppKit

// A nonactivating panel owns no keyboard or mouse input. All AppKit work runs
// on the main thread; newline-delimited updates arrive through the parent pipe.
final class SubtitleView: NSView {
    var text = ""
    var progress: CGFloat = 0
    let attributes: [NSAttributedString.Key: Any] = [
        .font: NSFont.systemFont(ofSize: 15, weight: .medium),
        .foregroundColor: NSColor.white
    ]
    override func draw(_ dirtyRect: NSRect) {
        NSColor.black.setFill()
        NSBezierPath(roundedRect: bounds, xRadius: 14, yRadius: 14).fill()
        let viewport = bounds.insetBy(dx: 20, dy: 0)
        NSGraphicsContext.saveGraphicsState()
        NSBezierPath(rect: viewport).addClip()
        let size = (text as NSString).size(withAttributes: attributes)
        let offset = max(0, size.width - viewport.width) * progress
        (text as NSString).draw(at: NSPoint(x: viewport.minX - offset,
                                          y: (bounds.height - size.height) / 2),
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
        let width = min(CGFloat(520), screen.frame.width - 32)
        let top = screen.safeAreaInsets.top > 0
            ? screen.frame.maxY - screen.safeAreaInsets.top
            : screen.visibleFrame.maxY
        panel.setFrame(NSRect(x: screen.frame.midX - width / 2,
                              y: top - 38, width: width, height: 38), display: true)
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
