import QtQuick
import QtQuick.Controls as Controls
import IshQt

Controls.ComboBox {
    id: control
    property color styleWindowColor: "#f2f2f7"
    implicitHeight: 40
    contentItem: Text { text: control.displayText; color: IOSPalette.text(control.styleWindowColor); font: control.font; verticalAlignment: Text.AlignVCenter; elide: Text.ElideRight }
    background: Rectangle { radius: 9; color: IOSPalette.elevatedSurface(control.styleWindowColor); border.width: control.activeFocus ? 2 : 1; border.color: control.activeFocus ? IOSPalette.blue : IOSPalette.separator(control.styleWindowColor) }
}
