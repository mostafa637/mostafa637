import QtQuick
import QtQuick.Controls as Controls
import IshQt

Controls.Popup {
    id: drawer
    property color styleWindowColor: "#f2f2f7"
    property int drawerWidth: Math.min(320, Math.max(240, parent ? parent.width * 0.82 : 280))
    property int edge: Qt.LeftEdge
    padding: 0
    modal: true
    dim: true
    closePolicy: Controls.Popup.CloseOnEscape | Controls.Popup.CloseOnPressOutside
    x: edge === Qt.RightEdge && parent ? parent.width - drawerWidth : 0
    y: 0
    width: drawerWidth
    height: parent ? parent.height : 0
    contentItem: Rectangle {
        color: IOSPalette.surface(drawer.styleWindowColor)
        border.color: IOSPalette.separator(drawer.styleWindowColor)
        border.width: 1
        Flickable {
            anchors.fill: parent
            anchors.margins: 12
            contentWidth: width
            contentHeight: drawerContent.implicitHeight
            clip: true
            Column { id: drawerContent; width: parent.width; spacing: 8 }
        }
    }
    default property alias items: drawerContent.data
}
