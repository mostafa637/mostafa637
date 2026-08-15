import QtQuick
import QtQuick.Controls as Controls
import QtQuick.Layouts
import IshQt

Controls.ToolButton {
    id: control
    property int index: 0
    property var tabBar: null
    property color styleWindowColor: tabBar ? tabBar.styleWindowColor : "#f2f2f7"
    Layout.fillWidth: true
    implicitHeight: 48
    checkable: true
    checked: tabBar ? tabBar.currentIndex === index : false
    onClicked: if (tabBar) tabBar.select(index)
    contentItem: Text { text: control.text; color: control.checked ? IOSPalette.blue : IOSPalette.secondaryText(control.styleWindowColor); font: control.font; font.pixelSize: 13; horizontalAlignment: Text.AlignHCenter; verticalAlignment: Text.AlignVCenter }
    background: Rectangle { color: "transparent"; Rectangle { width: parent.width; height: 2; anchors.bottom: parent.bottom; color: IOSPalette.blue; visible: control.checked } }
}
