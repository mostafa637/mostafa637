import QtQuick
import QtQuick.Layouts
import IshQt

Rectangle {
    id: bar
    property int currentIndex: 0
    property color styleWindowColor: "#f2f2f7"
    default property alias tabItems: tabs.data
    signal currentIndexChangedByUser(int index)
    implicitHeight: 50
    color: IOSPalette.surface(styleWindowColor)
    border.color: IOSPalette.separator(styleWindowColor)
    border.width: 1

    RowLayout {
        id: tabs
        anchors.fill: parent
        spacing: 0
    }

    function select(index) {
        currentIndex = index
        currentIndexChangedByUser(index)
    }
}
