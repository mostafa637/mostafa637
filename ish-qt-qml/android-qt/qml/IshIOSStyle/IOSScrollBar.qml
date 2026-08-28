import QtQuick
import QtQuick.Controls as Controls
import IshQt

Controls.ScrollBar {
    id: bar
    property color styleWindowColor: "#f2f2f7"
    implicitWidth: orientation === Qt.Vertical ? 7 : 0
    implicitHeight: orientation === Qt.Horizontal ? 7 : 0
    policy: Controls.ScrollBar.AsNeeded
    contentItem: Rectangle {
        implicitWidth: bar.orientation === Qt.Vertical ? 6 : 40
        implicitHeight: bar.orientation === Qt.Horizontal ? 6 : 40
        radius: width / 2
        color: bar.pressed ? IOSPalette.secondaryText(bar.styleWindowColor) : IOSPalette.separator(bar.styleWindowColor)
        opacity: bar.active ? 0.85 : 0.45
    }
    background: Item { }
}
