// Renders the extension icon set: Claude-orange rounded tile, white starburst,
// and a notification badge (white dot, top-right). Sizes 16/48/128.
// Usage: swift scripts/make-extension-icons.swift <outdir>
import AppKit

let outDir = CommandLine.arguments.count > 1 ? CommandLine.arguments[1] : "extension"

func drawIcon(size: CGFloat, withBadge: Bool) -> NSImage {
    let img = NSImage(size: NSSize(width: size, height: size))
    img.lockFocus()
    let ctx = NSGraphicsContext.current!.cgContext

    // Tile
    let tile = NSBezierPath(roundedRect: NSRect(x: 0, y: 0, width: size, height: size),
                            xRadius: size * 0.22, yRadius: size * 0.22)
    NSColor(calibratedRed: 0.85, green: 0.47, blue: 0.34, alpha: 1).setFill() // #D97757
    tile.fill()

    // Starburst: 8 capsules + hub, centered slightly low-left to make badge room
    let cx = size * (withBadge ? 0.46 : 0.5)
    let cy = size * (withBadge ? 0.46 : 0.5)
    let rOuter = size * 0.34
    let rayW = max(1.5, size * 0.07)
    NSColor(calibratedRed: 0.98, green: 0.96, blue: 0.94, alpha: 1).setFill()
    for i in 0..<8 {
        let angle = CGFloat(i) * .pi / 4
        ctx.saveGState()
        ctx.translateBy(x: cx, y: cy)
        ctx.rotate(by: angle)
        let ray = NSBezierPath(roundedRect: NSRect(x: -rayW/2, y: rOuter * 0.35, width: rayW, height: rOuter * 0.65),
                               xRadius: rayW/2, yRadius: rayW/2)
        ray.fill()
        ctx.restoreGState()
    }
    NSBezierPath(ovalIn: NSRect(x: cx - size*0.07, y: cy - size*0.07, width: size*0.14, height: size*0.14)).fill()

    // Notification badge: white ring + solid white dot, top-right
    if withBadge {
        let bd = size * 0.30
        let bx = size * 0.66, by = size * 0.66
        // ring (tile-colored gap so it pops)
        NSColor(calibratedRed: 0.85, green: 0.47, blue: 0.34, alpha: 1).setFill()
        NSBezierPath(ovalIn: NSRect(x: bx - bd*0.12, y: by - bd*0.12, width: bd*1.24, height: bd*1.24)).fill()
        NSColor.white.setFill()
        NSBezierPath(ovalIn: NSRect(x: bx, y: by, width: bd, height: bd)).fill()
    }

    img.unlockFocus()
    return img
}

for size in [16, 48, 128] {
    // 16px: skip the badge — it turns to mush at that scale
    let img = drawIcon(size: CGFloat(size), withBadge: size >= 48)
    guard let tiff = img.tiffRepresentation,
          let rep = NSBitmapImageRep(data: tiff),
          let png = rep.representation(using: .png, properties: [:]) else {
        fatalError("render failed at \(size)")
    }
    let path = "\(outDir)/icon\(size).png"
    try! png.write(to: URL(fileURLWithPath: path))
    print("wrote \(path)")
}
