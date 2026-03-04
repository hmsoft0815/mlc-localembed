import AppKit
import SwiftUI

class AppDelegate: NSObject, NSApplicationDelegate {
    var statusBarItem: NSStatusItem!
    let serviceName = "com.localembed.server"
    let plistPath = "~/Library/LaunchAgents/com.localembed.server.plist"

    func applicationDidFinishLaunching(_ notification: Notification) {
        statusBarItem = NSStatusBar.system.statusItem(withLength: NSStatusItem.variableLength)
        
        if let button = statusBarItem.button {
            button.image = NSImage(systemSymbolName: "bolt.fill", accessibilityDescription: "LocalEmbed")
        }
        
        setupMenu()
    }

    func setupMenu() {
        let menu = NSMenu()
        
        let status = isServiceRunning() ? "🟢 Running" : "🔴 Stopped"
        menu.addItem(NSMenuItem(title: "Status: \(status)", action: nil, keyEquivalent: ""))
        menu.addItem(NSMenuItem.separator())
        
        menu.addItem(NSMenuItem(title: "Start Server", action: #selector(startService), keyEquivalent: "s"))
        menu.addItem(NSMenuItem(title: "Stop Server", action: #selector(stopService), keyEquivalent: "x"))
        menu.addItem(NSMenuItem.separator())
        
        menu.addItem(NSMenuItem(title: "Open Logs", action: #selector(openLogs), keyEquivalent: "l"))
        menu.addItem(NSMenuItem(title: "Quit", action: #selector(NSApplication.terminate(_:)), keyEquivalent: "q"))
        
        statusBarItem.menu = menu
    }

    @objc func startService() {
        shell("launchctl load \(plistPath)")
        setupMenu()
    }

    @objc func stopService() {
        shell("launchctl unload \(plistPath)")
        setupMenu()
    }

    @objc func openLogs() {
        shell("open /usr/local/var/log/localembed.log")
    }

    func isServiceRunning() -> Bool {
        let output = shell("launchctl list | grep \(serviceName)")
        return !output.isEmpty
    }

    @discardableResult
    func shell(_ command: String) -> String {
        let task = Process()
        let pipe = Pipe()
        
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

// Main Entry Point
let app = NSApplication.shared
let delegate = AppDelegate()
app.delegate = delegate
app.run()
