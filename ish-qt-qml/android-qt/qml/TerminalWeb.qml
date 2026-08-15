import QtQuick
import QtWebView
import IshQt

Item {
    id: root
    property var terminalStyle: ({})
    property url pageUrl: ""
    signal ready()
    signal failed(string message)

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
    function setControlModifier(active) { }
    function sendAccessoryInput(value) { view.runJavaScript("window.ishSendInput && window.ishSendInput(" + JSON.stringify(value) + ")") }
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
                root.ready()
                root.focusTerminal()
            } else if (loadRequest.status === WebView.LoadFailedStatus) {
                root.failed("WebView failed to load: " + loadRequest.errorString)
            }
        }
    }
}
