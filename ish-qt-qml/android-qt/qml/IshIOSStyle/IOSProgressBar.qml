import QtQuick
import QtQuick.Controls as Controls
import IshQt

Controls.ProgressBar {
    id: control

    property color styleWindowColor: "#f2f2f7"

    implicitHeight: 8

    background: Rectangle {
        radius: 4
        color: IOSPalette.separator(control.styleWindowColor)
    }

    contentItem: Item {
        Rectangle {
            width: control.visualPosition * parent.width
            height: parent.height
            radius: 4
            color: IOSPalette.blue
        }
    }
}
