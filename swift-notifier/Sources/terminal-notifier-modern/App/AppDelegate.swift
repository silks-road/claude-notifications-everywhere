import AppKit
import UserNotifications

final class AppDelegate: NSObject, NSApplicationDelegate, UNUserNotificationCenterDelegate {

    private let actionExecutor: ActionExecuting

    init(actionExecutor: ActionExecuting = ActionExecutor()) {
        self.actionExecutor = actionExecutor
        super.init()
    }

    func applicationDidFinishLaunching(_ notification: Notification) {
        UNUserNotificationCenter.current().delegate = self
    }

    func userNotificationCenter(
        _ center: UNUserNotificationCenter,
        didReceive response: UNNotificationResponse,
        withCompletionHandler completionHandler: @escaping () -> Void
    ) {
        switch response.actionIdentifier {
        case "DISMISS", UNNotificationDismissActionIdentifier:
            break
        case "ALWAYS_ALLOW":
            executeFromUserInfo(response, key: "actionAlways")
        case "ALLOW_ONCE":
            executeFromUserInfo(response, key: "actionOnce")
        case "OPEN", UNNotificationDefaultActionIdentifier:
            let userInfo = response.notification.request.content.userInfo
            if let actionJSON = userInfo["action"] as? String,
               let action = ClickAction.fromJSON(actionJSON) {
                actionExecutor.execute(action)
            }
        default:
            let userInfo = response.notification.request.content.userInfo
            if let actionJSON = userInfo["action"] as? String,
               let action = ClickAction.fromJSON(actionJSON) {
                actionExecutor.execute(action)
            }
        }

        completionHandler()

        DispatchQueue.main.asyncAfter(deadline: .now() + 0.5) {
            NSApplication.shared.terminate(nil)
        }
    }

    private func executeFromUserInfo(_ response: UNNotificationResponse, key: String) {
        let userInfo = response.notification.request.content.userInfo
        if let actionJSON = userInfo[key] as? String,
           let action = ClickAction.fromJSON(actionJSON) {
            actionExecutor.execute(action)
        }
    }

    func userNotificationCenter(
        _ center: UNUserNotificationCenter,
        willPresent notification: UNNotification,
        withCompletionHandler completionHandler: @escaping (UNNotificationPresentationOptions) -> Void
    ) {
        if #available(macOS 11.0, *) {
            completionHandler([.banner, .sound])
        } else {
            completionHandler([.alert, .sound])
        }
    }
}
