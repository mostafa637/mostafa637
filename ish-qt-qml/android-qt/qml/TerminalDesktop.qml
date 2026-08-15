import QtQuick
import QtQuick.Controls
import IshQt

Rectangle {
    id: root

    property var terminalStyle: ({})
    property int overrideFontSize: 0
    property bool controlModifier: false
    property alias text: editor.text

    signal ready()
    signal failed(string message)

    function effectiveFontSize() {
        const base = Number(root.terminalStyle.fontSize || 14)
        return Math.max(6, Math.min(72, root.overrideFontSize > 0 ? root.overrideFontSize : base))
    }

    function setOverrideFontSize(value) {
        root.overrideFontSize = Math.max(0, Math.min(72, Number(value) || 0))
        Qt.callLater(updateTerminalSize)
    }

    function increaseFontSize() { root.setOverrideFontSize(root.effectiveFontSize() + 1) }
    function decreaseFontSize() { root.setOverrideFontSize(root.effectiveFontSize() - 1) }
    function resetFontSize() { root.setOverrideFontSize(0) }

    function focusTerminal() {
        editor.forceActiveFocus()
        root.scrollToBottom()
    }

    function setFocused(focused) {
        if (focused)
            root.focusTerminal()
        else
            editor.focus = false
    }

    function setControlModifier(active) {
        root.controlModifier = Boolean(active)
    }

    function controlCharacter(value) {
        if (!value || value.length === 0)
            return ""
        const character = value.charAt(0).toLowerCase()
        if (character === " " || character === "2") return "\u0000"
        if (character === "6") return "\u001e"
        if (character === "-") return "\u001f"
        const code = character.charCodeAt(0)
        if ((code >= 97 && code <= 122) || character === "@" || character === "^" ||
                character === "[" || character === "\\" || character === "]" || character === "_")
            return String.fromCharCode(code & 0x1f)
        return ""
    }

    function sendText(value) {
        if (!ishSession.alive || !value || value.length === 0)
            return
        ishSession.sendInput(value.replace(/\n/g, "\r"))
    }

    function sendAccessoryInput(value) {
        if (!value || value.length === 0 || !ishSession.alive)
            return
        if (root.controlModifier && value.length === 1) {
            const control = root.controlCharacter(value)
            root.controlModifier = false
            if (control.length > 0) {
                root.sendText(control)
                ishSession.controlModifierConsumed()
                root.focusTerminal()
                return
            }
        }
        root.controlModifier = false
        root.sendText(value)
        ishSession.controlModifierConsumed()
        root.focusTerminal()
    }

    function paste() {
        const value = platformServices.pasteText()
        if (value && value.length > 0)
            root.sendText(value)
        root.focusTerminal()
    }

    function hideKeyboard() {
        Qt.inputMethod.hide()
        root.focusTerminal()
    }

    function copySelection() { editor.copy() }

    function clearScrollback() {
        const value = editor.text
        const keep = Math.min(1024, value.length)
        editor.text = keep > 0 ? value.slice(value.length - keep) : ""
        editor.cursorPosition = editor.length
        root.scrollToBottom()
    }

    function scrollToBottom() {
        viewport.contentY = Math.max(0, viewport.contentHeight - viewport.height)
    }

    function scrollTo(top) {
        const value = Number(top)
        if (isFinite(value))
            viewport.contentY = Math.max(0, Math.min(value, Math.max(0, viewport.contentHeight - viewport.height)))
    }

    function updateTerminalSize() {
        if (!ishSession.alive)
            return
        const cellWidth = Math.max(7, editor.font.pixelSize * 0.62)
        const cellHeight = Math.max(12, editor.font.pixelSize * 1.35)
        ishSession.resize(Math.max(20, Math.floor(width / cellWidth)),
                          Math.max(4, Math.floor(height / cellHeight)))
    }

    color: root.terminalStyle.backgroundColor || "#000000"
    clip: true

    Flickable {
        id: viewport
        anchors.fill: parent
        clip: true
        boundsBehavior: Flickable.StopAtBounds
        contentWidth: Math.max(width, editor.contentWidth)
        contentHeight: Math.max(height, editor.contentHeight)

        TextEdit {
            id: editor
            width: viewport.width
            height: Math.max(viewport.height, contentHeight)
            color: root.terminalStyle.foregroundColor || "#f5f5f7"
            selectionColor: root.terminalStyle.selectionColor || "#3a78d4"
            selectedTextColor: root.terminalStyle.selectedTextColor || "#ffffff"
            font.family: root.terminalStyle.fontFamily || "Noto Sans Mono"
            font.pixelSize: root.effectiveFontSize()
            font.bold: false
            wrapMode: TextEdit.NoWrap
            readOnly: true
            selectByMouse: true
            cursorVisible: false
            activeFocusOnPress: false
            renderType: Text.NativeRendering

            Keys.onPressed: function(event) {
                let value = ""
                if (event.key === Qt.Key_Backspace) value = "\u0008"
                else if (event.key === Qt.Key_Return || event.key === Qt.Key_Enter) value = "\r"
                else if (event.key === Qt.Key_Tab) value = "\t"
                else if (event.key === Qt.Key_Escape) value = "\u001b"
                else if (event.key === Qt.Key_Left) value = "\u001b[D"
                else if (event.key === Qt.Key_Right) value = "\u001b[C"
                else if (event.key === Qt.Key_Up) value = "\u001b[A"
                else if (event.key === Qt.Key_Down) value = "\u001b[B"
                else if (event.text && !event.modifiers) value = event.text

                if (root.controlModifier && value.length === 1) {
                    const control = root.controlCharacter(value)
                    root.controlModifier = false
                    ishSession.controlModifierConsumed()
                    if (control.length > 0)
                        root.sendText(control)
                } else if (value.length > 0) {
                    root.sendText(value)
                    if (root.controlModifier) {
                        root.controlModifier = false
                        ishSession.controlModifierConsumed()
                    }
                }
                if (value.length > 0)
                    event.accepted = true
            }

            onContentHeightChanged: Qt.callLater(root.scrollToBottom)
        }
    }

    Connections {
        target: ishSession
        function onOutputReady(value) {
            editor.append(value)
            editor.cursorPosition = editor.length
            Qt.callLater(root.scrollToBottom)
        }
        function onStyleChanged(style) {
            root.terminalStyle = style
        }
        function onAliveChanged(alive) {
            if (alive)
                Qt.callLater(root.updateTerminalSize)
        }
    }

    Timer {
        interval: 0
        running: true
        repeat: false
        onTriggered: {
            root.ready()
            root.focusTerminal()
            root.updateTerminalSize()
        }
    }

    onWidthChanged: Qt.callLater(updateTerminalSize)
    onHeightChanged: Qt.callLater(updateTerminalSize)
}
