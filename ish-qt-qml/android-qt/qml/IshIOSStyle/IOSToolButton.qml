import QtQuick
import QtQuick.Controls as Controls
import IshQt

Controls.ToolButton {
    id: control
    property color styleWindowColor: Controls.ApplicationWindow.window ? Controls.ApplicationWindow.window.color : "#f2f2f7"
    implicitWidth: IOSMetrics.minimumTouchTarget
    implicitHeight: IOSMetrics.minimumTouchTarget
    padding: 4
    font.pixelSize: IOSMetrics.navigationButtonSize
    contentItem: Text {
        text: control.text
        color: control.enabled ? IOSPalette.blue : IOSPalette.secondaryText(control.styleWindowColor)
        font: control.font
        horizontalAlignment: Text.AlignHCenter
        verticalAlignment: Text.AlignVCenter
        elide: Text.ElideRight
    }
    background: Rectangle {
        radius: IOSMetrics.controlCornerRadius
        color: control.pressed ? IOSPalette.separator(control.styleWindowColor) : "transparent"
        border.width: control.pressed ? 1 : 0
        border.color: IOSPalette.separator(control.styleWindowColor)
    }
}
