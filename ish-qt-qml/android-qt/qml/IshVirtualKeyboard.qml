pragma ComponentBehavior: Bound
import QtQuick

Rectangle {
    id: root

    signal inputRequested(string value)
    signal dismissRequested()

    property bool darkMode: true
    property bool shifted: false
    property bool symbolMode: false
    property real panelPadding: 8
    property real keyGap: Math.max(4, Math.min(8, width * 0.009))
    property real rowGap: Math.max(5, Math.min(8, height * 0.02))
    property real keyHeight: Math.max(40, Math.min(62,
        (height - panelPadding * 2 - rowGap * 3) / 4))
    property color panelColor: darkMode ? "#1c1c1e" : "#d1d1d6"
    property color keyColor: darkMode ? "#5a5a5c" : "#ffffff"
    property color specialKeyColor: darkMode ? "#3a3a3c" : "#b8b8bd"
    property color pressedKeyColor: darkMode ? "#8e8e93" : "#aeb4be"
    property color keyTextColor: darkMode ? "#ffffff" : "#1c1c1e"
    property color specialTextColor: darkMode ? "#ffffff" : "#1c1c1e"
    property var letterRow1: ["q", "w", "e", "r", "t", "y", "u", "i", "o", "p"]
    property var letterRow2: ["a", "s", "d", "f", "g", "h", "j", "k", "l"]
    property var letterRow3: ["z", "x", "c", "v", "b", "n", "m"]
    property var symbolRow1: ["1", "2", "3", "4", "5", "6", "7", "8", "9", "0"]
    property var symbolRow2: ["-", "/", ":", ";", "(", ")", "$", "&", "@", "\""]
    property var symbolRow3: [".", ",", "?", "!", "'", "'", "#"]

    implicitHeight: 286
    color: panelColor
    // Keys are laid out wholly inside the panel. Avoid a clipped scenegraph
    // batch on Android; the emulator's RHI can turn clipped rounded delegates
    // into gray triangles.
    clip: false
    property bool flatAndroidDecorations: Qt.platform.os === "android"

    function unitWidth() {
        return Math.max(20, (width - panelPadding * 2 - keyGap * 9) / 10)
    }

    function emitText(value) {
        if (!value || value.length === 0)
            return
        inputRequested(value)
    }

    function emitLetter(value) {
        emitText(shifted ? value.toUpperCase() : value)
        if (shifted)
            shifted = false
    }

    component KeyCap: Rectangle {
        id: cap
        property string label: ""
        property string value: ""
        property bool special: false
        property real weight: 1
        signal activated()
        property bool pressedState: false

        radius: root.flatAndroidDecorations ? 0 : 7
        layer.enabled: root.flatAndroidDecorations
        layer.smooth: true
        color: pressedState ? root.pressedKeyColor : (special ? root.specialKeyColor : root.keyColor)
        border.width: root.flatAndroidDecorations ? 0 : 1
        border.color: root.darkMode ? "#6d6d70" : "#c7c7cc"
        opacity: enabled ? 1.0 : 0.65

        Text {
            anchors.fill: parent
            anchors.margins: 2
            text: cap.label
            color: cap.special ? root.specialTextColor : root.keyTextColor
            font.pixelSize: cap.special ? Math.max(15, Math.min(23, cap.height * 0.38))
                                       : Math.max(18, Math.min(30, cap.height * 0.48))
            font.bold: !cap.special
            horizontalAlignment: Text.AlignHCenter
            verticalAlignment: Text.AlignVCenter
            elide: Text.ElideNone
        }

        MouseArea {
            anchors.fill: parent
            onPressed: cap.pressedState = true
            onCanceled: cap.pressedState = false
            onReleased: {
                cap.pressedState = false
                if (containsMouse)
                    cap.activated()
            }
        }
    }

    Column {
        anchors.fill: parent
        anchors.margins: root.panelPadding
        spacing: root.rowGap

        Row {
            id: rowOne
            width: parent.width
            height: root.keyHeight
            spacing: root.keyGap

            Repeater {
                model: root.symbolMode ? root.symbolRow1 : root.letterRow1
                delegate: KeyCap {
                    required property string modelData
                    width: root.unitWidth()
                    height: root.keyHeight
                    label: root.symbolMode ? modelData : (root.shifted ? modelData.toUpperCase() : modelData.toUpperCase())
                    value: modelData
                    onActivated: root.symbolMode ? root.emitText(value) : root.emitLetter(value)
                }
            }
        }

        Row {
            id: rowTwo
            width: root.unitWidth() * 9 + root.keyGap * 8
            height: root.keyHeight
            anchors.horizontalCenter: parent.horizontalCenter
            spacing: root.keyGap

            Repeater {
                model: root.symbolMode ? root.symbolRow2 : root.letterRow2
                delegate: KeyCap {
                    required property string modelData
                    width: root.unitWidth()
                    height: root.keyHeight
                    label: root.symbolMode ? modelData : (root.shifted ? modelData.toUpperCase() : modelData.toUpperCase())
                    value: modelData
                    onActivated: root.symbolMode ? root.emitText(value) : root.emitLetter(value)
                }
            }
        }

        Row {
            id: rowThree
            width: root.unitWidth() * 9.5 + root.keyGap * 9
            height: root.keyHeight
            anchors.horizontalCenter: parent.horizontalCenter
            spacing: root.keyGap

            KeyCap {
                width: root.unitWidth() * 1.25
                height: root.keyHeight
                label: root.symbolMode ? "ABC" : "⇧"
                special: true
                onActivated: {
                    if (root.symbolMode)
                        root.symbolMode = false
                    else
                        root.shifted = !root.shifted
                }
            }

            Repeater {
                model: root.symbolMode ? root.symbolRow3 : root.letterRow3
                delegate: KeyCap {
                    required property string modelData
                    width: root.unitWidth()
                    height: root.keyHeight
                    label: root.symbolMode ? modelData : (root.shifted ? modelData.toUpperCase() : modelData.toUpperCase())
                    value: modelData
                    onActivated: root.symbolMode ? root.emitText(value) : root.emitLetter(value)
                }
            }

            KeyCap {
                width: root.unitWidth() * 1.25
                height: root.keyHeight
                label: "⌫"
                special: true
                onActivated: root.emitText("\u007f")
            }
        }

        Row {
            id: rowFour
            width: parent.width
            height: root.keyHeight
            spacing: root.keyGap

            KeyCap {
                width: (rowFour.width - root.keyGap * 2) * 2.35 / 10
                height: root.keyHeight
                label: root.symbolMode ? "ABC" : "123"
                special: true
                onActivated: root.symbolMode = !root.symbolMode
            }

            KeyCap {
                width: (rowFour.width - root.keyGap * 2) * 5.3 / 10
                height: root.keyHeight
                label: "space"
                special: true
                onActivated: root.emitText(" ")
            }

            KeyCap {
                width: (rowFour.width - root.keyGap * 2) * 2.35 / 10
                height: root.keyHeight
                label: "return"
                special: true
                onActivated: root.emitText("\r")
            }
        }
    }
}
