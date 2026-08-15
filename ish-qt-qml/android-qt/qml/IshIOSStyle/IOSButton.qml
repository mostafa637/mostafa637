import QtQuick
import QtQuick.Controls as Controls
import IshQt

Controls.Button {
    id: control
    property color styleWindowColor: Controls.ApplicationWindow.window ? Controls.ApplicationWindow.window.color : "#f2f2f7"
    implicitHeight: 42
    padding: 10
    font.pixelSize: 16
    contentItem: Text { text: control.text; color: control.enabled ? IOSPalette.blue : IOSPalette.secondaryText(control.styleWindowColor); font: control.font; horizontalAlignment: Text.AlignHCenter; verticalAlignment: Text.AlignVCenter; elide: Text.ElideRight }
    background: Rectangle { radius: 10; color: control.pressed ? IOSPalette.separator(control.styleWindowColor) : IOSPalette.elevatedSurface(control.styleWindowColor); border.width: 1; border.color: IOSPalette.separator(control.styleWindowColor); opacity: control.enabled ? 1 : .55 }
}
