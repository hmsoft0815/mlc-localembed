import AppKit
import SwiftUI
import Foundation

// MARK: - AppDelegate
class AppDelegate: NSObject, NSApplicationDelegate {
    
    var statusBarItem: NSStatusItem!
    var statusTimer: Timer?
    
    // Config
    let apiURL = "http://localhost:9142/api/health"
    let serviceName = "com.localembed.server"
    let plistPath = "~/Library/LaunchAgents/com.localembed.server.plist"
    
    // Status State
    var isServerReachable = false

    func applicationDidFinishLaunching(_ notification: Notification) {
        // Init Status Item
        statusBarItem = NSStatusBar.system.statusItem(withLength: NSStatusItem.variableLength)
        
        if let button = statusBarItem.button {
            button.image = NSImage(systemSymbolName: "bolt.fill", accessibilityDescription: "LocalEmbed")
            button.imagePosition = .imageLeft
        }
        
        // Initial Menu Build
        updateMenu()
        
        // Start Polling Timer (every 5 seconds)
        statusTimer = Timer.scheduledTimer(withTimeInterval: 5.0, repeats: true) { [weak self] _ in
            self?.checkServerHealth()
        }
        
        // Immediate check
        checkServerHealth()
    }

    func updateMenu() {
        let menu = NSMenu()
        
        // --- Status Section ---
        let statusText = isServerReachable ? "🟢 API Online" : "🔴 Offline / Loading"
        let statusItem = NSMenuItem(title: statusText, action: nil, keyEquivalent: "")
        statusItem.isEnabled = false
        menu.addItem(statusItem)
        
        menu.addItem(NSMenuItem.separator())
        
        // --- Control Section ---
        if !isServerReachable {
            menu.addItem(NSMenuItem(title: "Start Server", action: #selector(startService), keyEquivalent: "s"))
        } else {
            menu.addItem(NSMenuItem(title: "Stop Server", action: #selector(stopService), keyEquivalent: "x"))
        }
        
        menu.addItem(NSMenuItem.separator())
        
        // --- Tools Section ---
        menu.addItem(NSMenuItem(title: "Download Models...", action: #selector(downloadModels), keyEquivalent: "d"))
        menu.addItem(NSMenuItem(title: "Open Logs", action: #selector(openLogs), keyEquivalent: "l"))
        menu.addItem(NSMenuItem(title: "Check for Updates", action: #selector(checkUpdates), keyEquivalent: ""))
        
        menu.addItem(NSMenuItem.separator())
        
        // --- Quit ---
        menu.addItem(NSMenuItem(title: "Quit LocalEmbed", action: #selector(NSApplication.terminate(_:)), keyEquivalent: "q"))
        
        statusBarItem.menu = menu
    }

    // MARK: - Actions
    
    @objc func startService() {
        shell("launchctl load \(plistPath)")
        // Give it a second, then check
        DispatchQueue.main.asyncAfter(deadline: .now() + 1.5) {
            self.checkServerHealth()
        }
    }

    @objc func stopService() {
        shell("launchctl unload \(plistPath)")
        isServerReachable = false
        updateMenu()
    }

    @objc func openLogs() {
        shell("open /usr/local/var/log/localembed.log")
    }
    
    @objc func downloadModels() {
        // Opens a Terminal window to run the preloader interactively (important for HF_TOKEN input)
        let script = "tell application \"Terminal\" to do script \"/usr/local/bin/localembed-preloader\""
        shell("osascript -e '\(script)'")
        shell("osascript -e 'tell application \"Terminal\" to activate'")
    }
    
    @objc func checkUpdates() {
        if let url = URL(string: "https://github.com/mlcmcp/localembed/releases") {
            NSWorkspace.shared.open(url)
        }
    }

    // MARK: - Health Check Logic
    
    func checkServerHealth() {
        guard let url = URL(string: apiURL) else { return }
        
        let task = URLSession.shared.dataTask(with: url) { [weak self] _, response, error in
            let reachable = (error == nil && (response as? HTTPURLResponse)?.statusCode == 200)
            
            DispatchQueue.main.async {
                if self?.isServerReachable != reachable {
                    self?.isServerReachable = reachable
                    self?.updateMenu()
                    
                    // Update Icon color/symbol based on status
                    if let button = self?.statusBarItem.button {
                        button.contentFilters = reachable ? [] : [CIFilter(name: "CIColorControls", parameters: [kCIInputSaturationKey: 0])!]
                    }
                }
            }
        }
        task.resume()
    }

    // MARK: - Helper
    
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
        return String(data: data, encoding: .utf8) ?? ""
    }
}

// MARK: - Main Loop
let app = NSApplication.shared
let delegate = AppDelegate()
app.delegate = delegate
app.run()
