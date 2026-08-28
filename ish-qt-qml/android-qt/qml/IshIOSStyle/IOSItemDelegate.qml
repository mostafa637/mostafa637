import QtQuick
import QtQuick.Controls as Controls
import IshQt

Controls.ItemDelegate {
    id: control
    property color styleWindowColor: "#f2f2f7"
    implicitHeight: IOSMetrics.groupedRowHeight
    padding: 0
    leftPadding: IOSMetrics.tableHorizontalInset
    rightPadding: IOSMetrics.tableHorizontalInset
    contentItem: Text {
        text: control.text
        color: IOSPalette.text(control.styleWindowColor)
        font.pixelSize: IOSMetrics.rowLabelSize
        verticalAlignment: Text.AlignVCenter
        elide: Text.ElideRight
    }
    background: Rectangle {
        color: control.pressed || control.highlighted ? IOSPalette.separator(control.styleWindowColor) : "transparent"
        radius: 0
    }
}
