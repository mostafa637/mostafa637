import QtQuick
import QtQuick.Controls as Controls
import IshQt

Controls.TextArea {
    id: control
    property color styleWindowColor: "#f2f2f7"
    implicitHeight: 110
    color: IOSPalette.text(styleWindowColor)
    placeholderTextColor: IOSPalette.secondaryText(styleWindowColor)
    wrapMode: TextEdit.Wrap
    padding: 12
    background: Rectangle {
        radius: 10
        color: IOSPalette.elevatedSurface(control.styleWindowColor)
        border.width: control.activeFocus ? 2 : 1
        border.color: control.activeFocus ? IOSPalette.blue : IOSPalette.separator(control.styleWindowColor)
    }
}
