import QtQuick
import QtQuick.Controls as Controls
import IshQt

Controls.CheckBox {
    id: control
    property color styleWindowColor: "#f2f2f7"
    contentItem: Text { text: control.text; color: IOSPalette.text(control.styleWindowColor); font: control.font; leftPadding: control.indicator.width + 8; verticalAlignment: Text.AlignVCenter }
    indicator: Rectangle { implicitWidth: 24; implicitHeight: 24; x: control.leftPadding; y: (control.height-height)/2; radius: 6; color: control.checked ? IOSPalette.blue : "transparent"; border.width: 1; border.color: control.checked ? IOSPalette.blue : IOSPalette.separator(control.styleWindowColor); Text { anchors.centerIn: parent; text: "✓"; color: "white"; visible: control.checked } }
}
