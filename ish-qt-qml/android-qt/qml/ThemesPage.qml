import QtQuick
import IshQt

IOSPage {
    id: root

    signal editRequested(string themeName)

    header: IOSToolBar {
        title: "Themes"
        onBackClicked: root.closeRequested()
    }

    Flickable {
        id: flick
        anchors.fill: parent
        contentWidth: width
        contentHeight: contentColumn.implicitHeight + 36
        clip: true
        boundsBehavior: Flickable.StopAtBounds

        Column {
            id: contentColumn
            width: flick.width - 2 * root.contentInset
            x: root.contentInset
            y: 18
            spacing: 18

            IOSLabel {
                width: parent.width
                height: IOSMetrics.sectionHeaderHeight
                text: "Installed themes"
                color: IOSPalette.secondaryText(root.pageBackground)
                font.pixelSize: IOSMetrics.sectionLabelSize
                verticalAlignment: Text.AlignVCenter
            }

            Rectangle {
                width: parent.width
                height: Math.max(IOSMetrics.groupedRowHeight, themeList.contentHeight)
                radius: IOSMetrics.groupedCornerRadius
                color: IOSPalette.elevatedSurface(root.pageBackground)
                clip: true

                ListView {
                    id: themeList
                    anchors.fill: parent
                    model: themes.themeNames
                    clip: true
                    delegate: Item {
                        width: themeList.width
                        height: IOSMetrics.groupedRowHeight

                        Rectangle {
                            anchors.fill: parent
                            color: mouse.containsMouse ? IOSPalette.separator(root.pageBackground) : "transparent"
                        }

                        IOSLabel {
                            anchors.left: parent.left
                            anchors.leftMargin: IOSMetrics.tableHorizontalInset
                            anchors.right: chevron.left
                            anchors.rightMargin: 8
                            anchors.verticalCenter: parent.verticalCenter
                            text: modelData
                            elide: Text.ElideRight
                        }

                        Text {
                            id: chevron
                            anchors.right: parent.right
                            anchors.rightMargin: IOSMetrics.tableHorizontalInset
                            anchors.verticalCenter: parent.verticalCenter
                            text: "›"
                            color: IOSPalette.secondaryText(root.pageBackground)
                            font.pixelSize: 28
                            font.weight: Font.Light
                        }

                        Rectangle {
                            visible: index < themeList.count - 1
                            anchors.left: parent.left
                            anchors.right: parent.right
                            anchors.bottom: parent.bottom
                            height: 1
                            color: IOSPalette.separator(root.pageBackground)
                            opacity: 0.55
                        }

                        MouseArea {
                            id: mouse
                            anchors.fill: parent
                            hoverEnabled: true
                            onClicked: {
                                preferences.themeName = modelData
                                root.editRequested(modelData)
                            }
                        }
                    }
                }
            }
        }

        IOSScrollBar {
            anchors.right: parent.right
            anchors.top: parent.top
            anchors.bottom: parent.bottom
            orientation: Qt.Vertical
            styleWindowColor: root.pageBackground
            position: flick.visibleArea.yPosition
            size: flick.visibleArea.heightRatio
            onPositionChanged: if (pressed) flick.contentY = position * Math.max(0, flick.contentHeight - flick.height)
        }
    }
}
