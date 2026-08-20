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
    clip: false

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

    function roundedPath(ctx, x, y, w, h, radius) {
        var r = Math.min(radius, Math.min(w, h) / 2)
        ctx.beginPath()
        ctx.moveTo(x + r, y)
        ctx.lineTo(x + w - r, y)
        ctx.quadraticCurveTo(x + w, y, x + w, y + r)
        ctx.lineTo(x + w, y + h - r)
        ctx.quadraticCurveTo(x + w, y + h, x + w - r, y + h)
        ctx.lineTo(x + r, y + h)
        ctx.quadraticCurveTo(x, y + h, x, y + h - r)
        ctx.lineTo(x, y + r)
        ctx.quadraticCurveTo(x, y, x + r, y)
        ctx.closePath()
    }

    function drawKey(ctx, x, y, w, h, label, special) {
        ctx.fillStyle = special ? root.specialKeyColor : root.keyColor
        // A single Canvas surface avoids the per-delegate clipped batches that
        // become gray triangles with the Android emulator's Qt Quick RHI.
        if (Qt.platform.os === "android") {
            ctx.fillRect(x, y, w, h)
        } else {
            roundedPath(ctx, x, y, w, h, 7)
            ctx.fill()
        }
        ctx.fillStyle = special ? root.specialTextColor : root.keyTextColor
        ctx.font = (special ? "500 " : "700 ")
                + (special ? Math.max(15, Math.min(23, h * 0.38))
                           : Math.max(18, Math.min(30, h * 0.48))) + "px sans-serif"
        ctx.textAlign = "center"
        ctx.textBaseline = "middle"
        ctx.fillText(label, x + w / 2, y + h / 2 + 1)
    }

    function drawKeyboard(ctx) {
        ctx.clearRect(0, 0, width, height)
        ctx.fillStyle = root.panelColor
        ctx.fillRect(0, 0, width, height)

        var unit = root.unitWidth()
        var gap = root.keyGap
        var h = root.keyHeight
        var y = root.panelPadding
        var values = root.symbolMode ? root.symbolRow1 : root.letterRow1
        for (var i = 0; i < values.length; ++i)
            drawKey(ctx, root.panelPadding + i * (unit + gap), y, unit, h,
                    root.symbolMode ? values[i] : (root.shifted ? values[i].toUpperCase() : values[i].toUpperCase()), false)

        y += h + root.rowGap
        values = root.symbolMode ? root.symbolRow2 : root.letterRow2
        var row2Width = unit * 9 + gap * 8
        var row2X = (width - row2Width) / 2
        for (i = 0; i < values.length; ++i)
            drawKey(ctx, row2X + i * (unit + gap), y, unit, h,
                    root.symbolMode ? values[i] : (root.shifted ? values[i].toUpperCase() : values[i].toUpperCase()), false)

        y += h + root.rowGap
        var specialWidth = unit * 1.25
        var row3Width = specialWidth * 2 + unit * 7 + gap * 8
        var row3X = (width - row3Width) / 2
        drawKey(ctx, row3X, y, specialWidth, h, root.symbolMode ? "ABC" : "⇧", true)
        values = root.symbolMode ? root.symbolRow3 : root.letterRow3
        var x = row3X + specialWidth + gap
        for (i = 0; i < values.length; ++i) {
            drawKey(ctx, x, y, unit, h,
                    root.symbolMode ? values[i] : (root.shifted ? values[i].toUpperCase() : values[i].toUpperCase()), false)
            x += unit + gap
        }
        drawKey(ctx, x, y, specialWidth, h, "←", true)

        y += h + root.rowGap
        var row4Gap = gap * 2
        var row4Width = width - root.panelPadding * 2
        var leftWidth = (row4Width - row4Gap) * 2.35 / 10
        var spaceWidth = (row4Width - row4Gap) * 5.3 / 10
        var rightWidth = (row4Width - row4Gap) * 2.35 / 10
        x = root.panelPadding
        drawKey(ctx, x, y, leftWidth, h, root.symbolMode ? "ABC" : "123", true)
        x += leftWidth + gap
        drawKey(ctx, x, y, spaceWidth, h, "space", true)
        x += spaceWidth + gap
        drawKey(ctx, x, y, rightWidth, h, "return", true)
    }

    Canvas {
        id: keyboardCanvas
        anchors.fill: parent
        antialiasing: false
        onPaint: {
            var ctx = getContext("2d")
            root.drawKeyboard(ctx)
        }
    }

    // Transparent hit targets keep input handling in QML while the Canvas is
    // the only visual surface, preventing RHI batch corruption.
    component KeyCap: MouseArea {
        property string label: ""
        property string value: ""
        property bool special: false
        signal activated()
        hoverEnabled: false
        onClicked: activated()
    }

    Column {
        anchors.fill: parent
        anchors.margins: root.panelPadding
        spacing: root.rowGap
        z: 2

        Row {
            width: parent.width
            height: root.keyHeight
            spacing: root.keyGap
            Repeater {
                model: root.symbolMode ? root.symbolRow1 : root.letterRow1
                delegate: KeyCap {
                    required property string modelData
                    width: root.unitWidth()
                    height: root.keyHeight
                    value: modelData
                    onActivated: root.symbolMode ? root.emitText(value) : root.emitLetter(value)
                }
            }
        }

        Row {
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
                    value: modelData
                    onActivated: root.symbolMode ? root.emitText(value) : root.emitLetter(value)
                }
            }
        }

        Row {
            width: root.unitWidth() * 9.5 + root.keyGap * 8
            height: root.keyHeight
            anchors.horizontalCenter: parent.horizontalCenter
            spacing: root.keyGap

            KeyCap {
                width: root.unitWidth() * 1.25
                height: root.keyHeight
                value: root.symbolMode ? "ABC" : "shift"
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
                    value: modelData
                    onActivated: root.symbolMode ? root.emitText(value) : root.emitLetter(value)
                }
            }

            KeyCap {
                width: root.unitWidth() * 1.25
                height: root.keyHeight
                value: "backspace"
                onActivated: root.emitText("\u007f")
            }
        }

        Row {
            width: parent.width
            height: root.keyHeight
            spacing: root.keyGap

            KeyCap {
                width: (parent.width - root.keyGap * 2) * 2.35 / 10
                height: root.keyHeight
                value: root.symbolMode ? "ABC" : "123"
                onActivated: root.symbolMode = !root.symbolMode
            }
            KeyCap {
                width: (parent.width - root.keyGap * 2) * 5.3 / 10
                height: root.keyHeight
                value: "space"
                onActivated: root.emitText(" ")
            }
            KeyCap {
                width: (parent.width - root.keyGap * 2) * 2.35 / 10
                height: root.keyHeight
                value: "return"
                onActivated: root.emitText("\r")
            }
        }
    }

    onWidthChanged: keyboardCanvas.requestPaint()
    onHeightChanged: keyboardCanvas.requestPaint()
    onDarkModeChanged: keyboardCanvas.requestPaint()
    onShiftedChanged: keyboardCanvas.requestPaint()
    onSymbolModeChanged: keyboardCanvas.requestPaint()
}
