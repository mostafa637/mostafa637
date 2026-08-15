import QtQuick
import QtQuick.Controls as Controls
import IshQt

Controls.Slider {
    id: control

    property color styleWindowColor: "#f2f2f7"

    implicitHeight: 36

    background: Rectangle {
        x: 0
        y: control.height / 2 - 2
        width: control.width
        height: 4
        radius: 2
        color: IOSPalette.separator(control.styleWindowColor)
    }

    handle: Rectangle {
        x: control.visualPosition * (control.width - width)
        y: control.height / 2 - height / 2
        width: 22
        height: 22
        radius: 11
        color: IOSPalette.blue
    }
}
