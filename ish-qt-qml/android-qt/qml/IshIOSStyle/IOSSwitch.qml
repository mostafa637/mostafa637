import QtQuick
import QtQuick.Controls as Controls
import IshQt

Controls.Switch {
    id: control
    property color styleWindowColor: "#f2f2f7"
    indicator: Rectangle { implicitWidth: 50; implicitHeight: 30; x: control.leftPadding; y: (control.height-height)/2; radius: 15; color: control.checked ? IOSPalette.green : IOSPalette.separator(control.styleWindowColor); Rectangle { width: 26; height: 26; radius: 13; y: 2; x: control.checked ? parent.width-width-2 : 2; color: "white" } }
    contentItem: Text { text: control.text; color: IOSPalette.text(control.styleWindowColor); font: control.font; leftPadding: control.indicator.width + 10; verticalAlignment: Text.AlignVCenter }
}
