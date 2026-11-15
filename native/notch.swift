import AppKit
import CoreGraphics

// UUID identifies the display across arrangement changes and reconnects;
// NSScreen array indices and session display numbers are not saved.
func displayIdentity(_ screen: NSScreen) -> String {
    let number = screen.deviceDescription[NSDeviceDescriptionKey("NSScreenNumber")] as? NSNumber
    guard let number = number,
          let uuid = CGDisplayCreateUUIDFromDisplayID(number.uint32Value)?.takeRetainedValue() else { return "" }
    return CFUUIDCreateString(nil, uuid) as String
}
func displayList() -> [[String: Any]] {
    return NSScreen.screens.compactMap { screen in
        let id = displayIdentity(screen)
        guard !id.isEmpty else { return nil }
        let number = screen.deviceDescription[NSDeviceDescriptionKey("NSScreenNumber")] as? NSNumber
        let displayID = number?.uint32Value ?? 0
        return ["id": id, "name": screen.localizedName,
                "builtIn": CGDisplayIsBuiltin(displayID) != 0,
                "width": Int(CGDisplayPixelsWide(displayID)),
                "height": Int(CGDisplayPixelsHigh(displayID))]
    }
}

// A nonactivating panel owns no keyboard or mouse input. All AppKit work runs
// on the main thread; newline-delimited updates arrive through the parent pipe.
final class SubtitleView: NSView {
    var text = ""
    var progress: CGFloat = 0
    var attachedToNotch = false
    let subtitleHeight: CGFloat = 32
    let transitionHeight: CGFloat = 6
    var connectionWidth: CGFloat = 180
    let attributes: [NSAttributedString.Key: Any] = [
        .font: NSFont.systemFont(ofSize: 14, weight: .medium),
        .foregroundColor: NSColor.white
    ]
    override func draw(_ dirtyRect: NSRect) {
        NSColor.black.setFill()
        if attachedToNotch {
            // Keep the transition shallow and straight. The two-pixel turns
            // soften the diagonal without creating the old flared shoulders.
            let radius: CGFloat = 11
            let neckLeft = (bounds.width - connectionWidth) / 2
            let neckRight = bounds.width - neckLeft
            let shoulderY = subtitleHeight
            let shape = NSBezierPath()
            shape.move(to: NSPoint(x: neckLeft, y: bounds.maxY))
            shape.line(to: NSPoint(x: neckRight, y: bounds.maxY))
            shape.line(to: NSPoint(x: bounds.maxX - 2, y: shoulderY + 1))
            shape.curve(to: NSPoint(x: bounds.maxX, y: shoulderY - 1),
                        controlPoint1: NSPoint(x: bounds.maxX, y: shoulderY + 0.5),
                        controlPoint2: NSPoint(x: bounds.maxX, y: shoulderY))
            shape.line(to: NSPoint(x: bounds.maxX, y: radius))
            shape.curve(to: NSPoint(x: bounds.maxX - radius, y: 0),
                        controlPoint1: NSPoint(x: bounds.maxX, y: radius * 0.448),
                        controlPoint2: NSPoint(x: bounds.maxX - radius * 0.448, y: 0))
            shape.line(to: NSPoint(x: radius, y: 0))
            shape.curve(to: NSPoint(x: 0, y: radius),
                        controlPoint1: NSPoint(x: radius * 0.448, y: 0),
                        controlPoint2: NSPoint(x: 0, y: radius * 0.448))
            shape.line(to: NSPoint(x: 0, y: shoulderY - 1))
            shape.curve(to: NSPoint(x: 2, y: shoulderY + 1),
                        controlPoint1: NSPoint(x: 0, y: shoulderY),
                        controlPoint2: NSPoint(x: 0, y: shoulderY + 0.5))
            shape.line(to: NSPoint(x: neckLeft, y: bounds.maxY))
            shape.close()
            shape.fill()
        } else {
            NSBezierPath(roundedRect: bounds, xRadius: 14, yRadius: 14).fill()
        }
        let textArea = NSRect(x: 0, y: 0, width: bounds.width, height: subtitleHeight)
        let viewport = textArea.insetBy(dx: 16, dy: 0)
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
    var selectedDisplay = "auto"
    var activeDisplay = ""
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
            self?.view.needsDisplay = true
            if let self = self, !self.view.text.isEmpty { self.panel.orderFrontRegardless() }
        }
    }
    func layout() {
        // Honor the saved display UUID. If absent, temporarily use automatic
        // selection without replacing it, so reconnecting restores the choice.
        let screens = NSScreen.screens
        let chosen = selectedDisplay == "auto" ? nil
            : screens.first(where: { displayIdentity($0) == selectedDisplay })
        guard let screen = chosen ?? screens.first(where: { $0.safeAreaInsets.top > 0 })
                ?? screens.first else { return }
        activeDisplay = displayIdentity(screen)
        let inset = screen.safeAreaInsets.top
        view.attachedToNotch = inset > 0
        if inset > 0 {
            // Derive the camera housing width from the two usable top regions.
            // Keep the expanded island compact, with enough room for one line.
            let left = screen.auxiliaryTopLeftArea
            let right = screen.auxiliaryTopRightArea
            let notchWidth = (left != nil && right != nil)
                ? max(0, right!.minX - left!.maxX) : 200
            let width = min(max(260, notchWidth + 56), screen.frame.width - 32)
            // A tiny overlap joins the black housing without painting either
            // menu-bar shoulder. Inset the neck to meet its rounded lower edge.
            view.connectionWidth = min(width - 32, max(100, notchWidth - 20))
            let height = view.transitionHeight + view.subtitleHeight
            let top = screen.frame.maxY - inset + 2
            panel.setFrame(NSRect(x: screen.frame.midX - width / 2,
                                  y: top - height,
                                  width: width, height: height), display: true)
        } else {
            let width = min(CGFloat(400), screen.frame.width - 32)
            panel.setFrame(NSRect(x: screen.frame.midX - width / 2,
                                  y: screen.visibleFrame.maxY - view.subtitleHeight,
                                  width: width, height: view.subtitleHeight), display: true)
        }
    }
    func update(_ message: [String: Any]) -> [String: Any] {
        if let display = message["displayId"] as? String { selectedDisplay = display }
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
                "frame": NSStringFromRect(panel.frame), "displayId": activeDisplay,
                "requestedDisplayId": selectedDisplay]
    }
}

let app = NSApplication.shared
app.setActivationPolicy(.accessory)
if CommandLine.arguments.contains("--screens") {
    let data = try JSONSerialization.data(withJSONObject: displayList())
    FileHandle.standardOutput.write(data)
    exit(0)
}
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
