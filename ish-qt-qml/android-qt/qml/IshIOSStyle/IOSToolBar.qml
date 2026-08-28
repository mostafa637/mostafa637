import QtQuick
import QtQuick.Controls as Controls
import IshQt

Rectangle {
    id: bar
    property string title: ""
    property string backText: "Back"
    property bool showBackButton: true
    property color styleWindowColor: parent && parent.pageBackground ? parent.pageBackground : "#f2f2f7"
    signal backClicked()
    default property alias toolItems: actions.data

    implicitHeight: IOSMetrics.navigationBarHeight
    height: IOSMetrics.navigationBarHeight
    color: IOSPalette.elevatedSurface(styleWindowColor)
    border.color: IOSPalette.separator(styleWindowColor)
    border.width: 1

    IOSToolButton {
        id: backButton
        objectName: "navigationBackButton"
        anchors.left: parent.left
        anchors.leftMargin: 4
        anchors.verticalCenter: parent.verticalCenter
        implicitWidth: Math.max(IOSMetrics.minimumTouchTarget, 58)
        implicitHeight: IOSMetrics.minimumTouchTarget
        visible: bar.showBackButton
        text: "‹ " + bar.backText
        font.pixelSize: IOSMetrics.navigationButtonSize
        onClicked: bar.backClicked()
    }

    IOSLabel {
        id: titleLabel
        anchors.centerIn: parent
        width: Math.min(parent.width - (backButton.visible ? backButton.width : 0) - actions.width - 32, parent.width - 32)
        text: bar.title
        font.bold: true
        font.pixelSize: IOSMetrics.navigationTitleSize
        horizontalAlignment: Text.AlignHCenter
        verticalAlignment: Text.AlignVCenter
        elide: Text.ElideMiddle
    }

    Row {
        id: actions
        anchors.right: parent.right
        anchors.rightMargin: 8
        anchors.verticalCenter: parent.verticalCenter
        spacing: 2
        height: IOSMetrics.minimumTouchTarget
    }
}
