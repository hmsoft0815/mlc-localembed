import AppKit
import SwiftUI

// AppDelegate entspricht der klassischen @interface / @implementation AppDelegate : NSObject <NSApplicationDelegate>
class AppDelegate: NSObject, NSApplicationDelegate {
    
    // NSStatusItem ist das Objekt für das Icon in der Menüleiste
    var statusBarItem: NSStatusItem!
    
    // Konstanten (let) statt #define oder static NSString
    let serviceName = "com.localembed.server"
    let plistPath = "~/Library/LaunchAgents/com.localembed.server.plist"

    // Entspricht - (void)applicationDidFinishLaunching:(NSNotification *)aNotification
    func applicationDidFinishLaunching(_ notification: Notification) {
        // Erstellt das Item in der System-Statusleiste (variable Breite)
        statusBarItem = NSStatusBar.system.statusItem(withLength: NSStatusItem.variableLength)
        
        if let button = statusBarItem.button {
            // SF Symbols (System-Icons) nutzen (neu in macOS 11+)
            button.image = NSImage(systemSymbolName: "bolt.fill", accessibilityDescription: "LocalEmbed")
        }
        
        // Menü initialisieren
        setupMenu()
    }

    func setupMenu() {
        // Entspricht [[NSMenu alloc] init]
        let menu = NSMenu()
        
        // Ternärer Operator wie in Obj-C
        let status = isServiceRunning() ? "🟢 Running" : "🔴 Stopped"
        
        // Entspricht [menu addItemWithTitle:...]
        menu.addItem(NSMenuItem(title: "Status: \(status)", action: nil, keyEquivalent: ""))
        menu.addItem(NSMenuItem.separator())
        
        // @objc func markiert Methoden, die über den alten Obj-C Selector-Mechanismus aufrufbar sind
        // keyEquivalent ist der Shortcut (z.B. "s" für Cmd+S)
        menu.addItem(NSMenuItem(title: "Start Server", action: #selector(startService), keyEquivalent: "s"))
        menu.addItem(NSMenuItem(title: "Stop Server", action: #selector(stopService), keyEquivalent: "x"))
        menu.addItem(NSMenuItem.separator())
        
        menu.addItem(NSMenuItem(title: "Open Logs", action: #selector(openLogs), keyEquivalent: "l"))
        
        // NSApplication.terminate ist der Standard-Exit
        menu.addItem(NSMenuItem(title: "Quit", action: #selector(NSApplication.terminate(_:)), keyEquivalent: "q"))
        
        statusBarItem.menu = menu
    }

    // Methoden mit @objc sind für [target performSelector:...] sichtbar
    @objc func startService() {
        shell("launchctl load \(plistPath)")
        setupMenu() // Menü neu zeichnen um Status zu aktualisieren
    }

    @objc func stopService() {
        shell("launchctl unload \(plistPath)")
        setupMenu()
    }

    @objc func openLogs() {
        shell("open /usr/local/var/log/localembed.log")
    }

    func isServiceRunning() -> Bool {
        // Prüft ob der Dienst in der launchctl Liste auftaucht
        let output = shell("launchctl list | grep \(serviceName)")
        return !output.isEmpty
    }

    // Hilfsfunktion zum Ausführen von Shell-Befehlen
    // Entspricht einer Kombination aus NSTask und NSPipe
    @discardableResult // Verhindert Warnung, wenn Rückgabewert ignoriert wird
    func shell(_ command: String) -> String {
        let task = Process() // Früher NSTask
        let pipe = Pipe()    // Früher NSPipe
        
        task.standardOutput = pipe
        task.standardError = pipe
        task.arguments = ["-c", command]
        task.launchPath = "/bin/bash"
        task.launch()
        
        let data = pipe.fileHandleForReading.readDataToEndOfFile()
        let output = String(data: data, encoding: .utf8) ?? ""
        return output
    }
}

// Main Entry Point (in Swift oft ohne explizite main.m)
let app = NSApplication.shared
let delegate = AppDelegate()
app.delegate = delegate
app.run()
