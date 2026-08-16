import QtQuick

Rectangle {
    id: root

    signal directionRequested(string escapeSequence)

    property bool darkMode: false
    property color normalColor: "#ffffff"
    property color pressedColor: "#b5b5b5"
    property color foreground: "#111111"
    property bool selected: false
    property int buttonSize: 48
    property string direction: ""
    property real startX: 0
    property real startY: 0
    property int activationDistance: 20

    implicitWidth: root.buttonSize
    implicitHeight: root.buttonSize
    radius: 9
    color: root.selected ? root.pressedColor : root.normalColor

    function sequenceFor(value) {
        if (value === "up") return "\u001b[A"
        if (value === "down") return "\u001b[B"
        if (value === "left") return "\u001b[D"
        if (value === "right") return "\u001b[C"
        return ""
    }

    function chooseDirection(x, y) {
        const dx = x - root.startX
        const dy = y - root.startY
        if (Math.hypot(dx, dy) < root.activationDistance)
            return false
        const nextDirection = Math.abs(dx) > Math.abs(dy)
                ? (dx > 0 ? "right" : "left")
                : (dy > 0 ? "down" : "up")
        if (nextDirection !== root.direction) {
            root.direction = nextDirection
            root.emitDirection()
            return true
        }
        return false
    }

    function directionAtPoint(x, y) {
        const dx = x - width / 2
        const dy = y - height / 2
        return Math.abs(dx) > Math.abs(dy)
                ? (dx >= 0 ? "right" : "left")
                : (dy >= 0 ? "down" : "up")
    }

    function emitDirection() {
        const value = root.sequenceFor(root.direction)
        if (value.length > 0)
            root.directionRequested(value)
    }

    Timer {
        id: firstRepeat
        interval: 500
        repeat: false
        onTriggered: {
            if (root.selected && root.direction.length > 0) {
                root.emitDirection()
                repeating.start()
            }
        }
    }

    Timer {
        id: repeating
        interval: 100
        repeat: true
        onTriggered: if (root.selected && root.direction.length > 0) root.emitDirection()
    }

    MouseArea {
        anchors.fill: parent
        preventStealing: true
        hoverEnabled: false

        onPressed: function(mouse) {
            root.startX = mouse.x
            root.startY = mouse.y
            root.direction = ""
            root.selected = true
            firstRepeat.stop()
            repeating.stop()
        }

        onPositionChanged: function(mouse) {
            if (pressed)
                root.chooseDirection(mouse.x, mouse.y)
            if (root.direction.length > 0 && !firstRepeat.running && !repeating.running)
                firstRepeat.start()
        }

        onReleased: function(mouse) {
            // A direct tap on a quadrant is a valid arrow press. The old
            // implementation required a 20px drag, so ordinary taps looked
            // dead on touch screens.
            if (root.direction.length === 0) {
                root.direction = root.directionAtPoint(mouse.x, mouse.y)
                root.emitDirection()
            }
            root.selected = false
            root.direction = ""
            firstRepeat.stop()
            repeating.stop()
        }

        onCanceled: {
            root.selected = false
            root.direction = ""
            firstRepeat.stop()
            repeating.stop()
        }
    }

    Image {
        width: root.buttonSize * 0.44
        height: root.buttonSize * 0.44
        anchors.horizontalCenter: parent.horizontalCenter
        anchors.bottom: parent.verticalCenter
        anchors.bottomMargin: -root.buttonSize * 0.06
        opacity: root.selected && root.direction !== "up" ? 0.25 : 1
        source: "qrc:/ish-assets/ui/icons/arrow-up-" + (root.darkMode ? "dark" : "light") + ".png"
        sourceSize: Qt.size(24, 24)
        fillMode: Image.PreserveAspectFit
        smooth: true
    }
    Image {
        width: root.buttonSize * 0.44
        height: root.buttonSize * 0.44
        anchors.horizontalCenter: parent.horizontalCenter
        anchors.top: parent.verticalCenter
        anchors.topMargin: -root.buttonSize * 0.06
        opacity: root.selected && root.direction !== "down" ? 0.25 : 1
        source: "qrc:/ish-assets/ui/icons/arrow-down-" + (root.darkMode ? "dark" : "light") + ".png"
        sourceSize: Qt.size(24, 24)
        fillMode: Image.PreserveAspectFit
        smooth: true
    }
    Image {
        width: root.buttonSize * 0.44
        height: root.buttonSize * 0.44
        anchors.right: parent.horizontalCenter
        anchors.rightMargin: -root.buttonSize * 0.06
        anchors.verticalCenter: parent.verticalCenter
        opacity: root.selected && root.direction !== "left" ? 0.25 : 1
        source: "qrc:/ish-assets/ui/icons/arrow-left-" + (root.darkMode ? "dark" : "light") + ".png"
        sourceSize: Qt.size(24, 24)
        fillMode: Image.PreserveAspectFit
        smooth: true
    }
    Image {
        width: root.buttonSize * 0.44
        height: root.buttonSize * 0.44
        anchors.left: parent.horizontalCenter
        anchors.leftMargin: -root.buttonSize * 0.06
        anchors.verticalCenter: parent.verticalCenter
        opacity: root.selected && root.direction !== "right" ? 0.25 : 1
        source: "qrc:/ish-assets/ui/icons/arrow-right-" + (root.darkMode ? "dark" : "light") + ".png"
        sourceSize: Qt.size(24, 24)
        fillMode: Image.PreserveAspectFit
        smooth: true
    }

    Rectangle {
        anchors.left: parent.left
        anchors.right: parent.right
        anchors.bottom: parent.bottom
        height: 2
        radius: 1
        color: root.selected ? "#55000000" : "#66000000"
    }
}

