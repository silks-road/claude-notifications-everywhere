import Foundation
import UserNotifications

enum NotificationCategory {

    static let categoryIdentifier = "CLAUDE_NOTIFICATION"
    static let approvalCategoryIdentifier = "CLAUDE_APPROVAL"

    static func register() {
        let openAction = UNNotificationAction(
            identifier: "OPEN",
            title: "Open",
            options: [.foreground]
        )

        let dismissAction = UNNotificationAction(
            identifier: "DISMISS",
            title: "Dismiss",
            options: [.destructive]
        )

        let category = UNNotificationCategory(
            identifier: categoryIdentifier,
            actions: [openAction, dismissAction],
            intentIdentifiers: [],
            options: []
        )

        // Approval notifications: primary button answers the pending
        // permission request; the dropdown offers a one-time grant.
        let alwaysAction = UNNotificationAction(
            identifier: "ALWAYS_ALLOW",
            title: "Always allow",
            options: []
        )
        let onceAction = UNNotificationAction(
            identifier: "ALLOW_ONCE",
            title: "Allow once",
            options: []
        )
        let approvalCategory = UNNotificationCategory(
            identifier: approvalCategoryIdentifier,
            actions: [alwaysAction, onceAction, openAction, dismissAction],
            intentIdentifiers: [],
            options: []
        )

        UNUserNotificationCenter.current().setNotificationCategories([category, approvalCategory])
    }
}
