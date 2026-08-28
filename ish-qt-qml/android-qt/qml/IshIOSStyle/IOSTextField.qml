import QtQuick
import QtQuick.Controls as Controls
import IshQt

Controls.TextField {
    id: control
    property color styleWindowColor: "#f2f2f7"
    implicitHeight: 40
    color: IOSPalette.text(styleWindowColor)
    placeholderTextColor: IOSPalette.secondaryText(styleWindowColor)
    background: Rectangle { radius: 9; color: IOSPalette.elevatedSurface(control.styleWindowColor); border.width: control.activeFocus ? 2 : 1; border.color: control.activeFocus ? IOSPalette.blue : IOSPalette.separator(control.styleWindowColor) }
}
