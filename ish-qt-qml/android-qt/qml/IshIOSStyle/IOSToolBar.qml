import QtQuick
import QtQuick.Layouts
import IshQt

Rectangle {
    id: bar
    property string title: ""
    property color styleWindowColor: parent && parent.pageBackground ? parent.pageBackground : "#f2f2f7"
    signal backClicked()
    default property alias toolItems: actions.data
    implicitHeight: 48
    color: IOSPalette.surface(styleWindowColor)
    border.color: IOSPalette.separator(styleWindowColor)
    border.width: 1
    RowLayout {
        anchors.fill: parent
        anchors.leftMargin: 6
        anchors.rightMargin: 6
        spacing: 4
        IOSToolButton { text: "‹  Back"; visible: bar.title.length > 0; onClicked: bar.backClicked() }
        IOSLabel { Layout.fillWidth: true; text: bar.title; font.bold: true; font.pixelSize: 17; horizontalAlignment: Text.AlignHCenter }
        RowLayout { id: actions; spacing: 2 }
    }
}
