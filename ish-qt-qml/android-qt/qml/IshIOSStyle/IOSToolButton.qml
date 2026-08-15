import QtQuick
import QtQuick.Controls as Controls
import IshQt

Controls.ToolButton {
    id: control
    property color styleWindowColor: Controls.ApplicationWindow.window ? Controls.ApplicationWindow.window.color : "#f2f2f7"
    implicitWidth: 42
    implicitHeight: 40
    font.pixelSize: 16
    contentItem: Text { text: control.text; color: control.enabled ? IOSPalette.blue : IOSPalette.secondaryText(control.styleWindowColor); font: control.font; horizontalAlignment: Text.AlignHCenter; verticalAlignment: Text.AlignVCenter; elide: Text.ElideRight }
    background: Rectangle { radius: 9; color: control.pressed ? IOSPalette.separator(control.styleWindowColor) : "transparent"; border.width: control.pressed ? 1 : 0; border.color: IOSPalette.separator(control.styleWindowColor) }
}
