import QtQuick
import QtWebView
import IshQt

Item {
    id: root

    property var terminalStyle: ({})
    property url pageUrl: ""
    property bool controlModifier: false
    signal ready()
    signal failed(string message)
    signal settingsRequested()
    property bool htmlKeyboardEnabled: Qt.platform.os === "android"

    function loadPage(url) {
        if (!url || String(url).length === 0) {
            failed("Terminal URL is empty")
            return
        }
        pageUrl = url
        view.url = url
    }

    function focusTerminal() { view.forceActiveFocus() }
    function setFocused(focused) { if (focused) focusTerminal() }

    function controlCharacter(value) {
        if (!value || value.length !== 1)
            return ""
        const character = value.toLowerCase()
        if (character === " " || character === "2") return "\u0000"
        if (character === "6") return "\u001e"
        if (character === "-") return "\u001f"
        const code = character.charCodeAt(0)
        if ((code >= 97 && code <= 122) || character === "@" || character === "^" ||
                character === "[" || character === "\\" || character === "]" || character === "_")
            return String.fromCharCode(code & 0x1f)
        return ""
    }

    function setControlModifier(active) {
        root.controlModifier = Boolean(active)
        view.runJavaScript("window.ishSetControlModifier && window.ishSetControlModifier(" +
                           JSON.stringify(root.controlModifier) + ")")
    }

    function setVirtualKeyboardVisible(visible) {
        if (!root.htmlKeyboardEnabled)
            return
        view.runJavaScript("window.ishSetVirtualKeyboardVisible && window.ishSetVirtualKeyboardVisible(" +
                           JSON.stringify(Boolean(visible)) + ")")
    }

    function sendAccessoryInput(value) {
        if (!value || value.length === 0)
            return
        let input = value
        if (root.controlModifier) {
            if (value.length === 1) {
                const control = root.controlCharacter(value)
                if (control.length > 0)
                    input = control
            }
            // Control is a one-shot modifier. Consume it even when the
            // selected accessory value is an escape sequence such as an
            // arrow key; otherwise it remains latched for the next key.
            root.controlModifier = false
        }
        view.runJavaScript("window.ishSendInput && window.ishSendInput(" + JSON.stringify(input) + ")")
        root.focusTerminal()
    }

    function paste() { view.runJavaScript("window.ishPaste && window.ishPaste()") }
    function hideKeyboard() { Qt.inputMethod.hide() }
    function increaseFontSize() { view.runJavaScript("window.ishFontStep && window.ishFontStep(1)") }
    function decreaseFontSize() { view.runJavaScript("window.ishFontStep && window.ishFontStep(-1)") }
    function resetFontSize() { view.runJavaScript("window.ishFontReset && window.ishFontReset()") }
    function clearScrollback() { view.runJavaScript("window.ishClear && window.ishClear()") }
    function copySelection() { view.runJavaScript("window.ishCopy && window.ishCopy()") }

    WebView {
        id: view
        anchors.fill: parent
        visible: true
        focus: true
        onLoadingChanged: function(loadRequest) {
            if (loadRequest.status === WebView.LoadSucceededStatus) {
                // Keep the WebView lifecycle one-way.  Qt WebView callbacks
                // scheduled from QML can outlive the Chromium surface on
                // Android and trigger a QV4 property-lookup crash.
                root.ready()
                root.focusTerminal()
                root.setVirtualKeyboardVisible(true)
            } else if (loadRequest.status === WebView.LoadFailedStatus) {
                root.failed("WebView failed to load: " + loadRequest.errorString)
            }
        }
        onTitleChanged: {
            // HTML toolbar actions cross the WebView boundary through the
            // title property.  This is synchronous and one-shot, unlike the
            // old runJavaScript polling callback that could outlive WebView.
            if (title.indexOf("ish-settings:") === 0)
                root.settingsRequested()
        }
    }
}
