import QtQuick
import IshQt

IOSPage {
    id: root

    header: IOSToolBar {
        title: "Filesystems"
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
                text: "Bundled root filesystem"
                color: IOSPalette.secondaryText(root.pageBackground)
                font.pixelSize: IOSMetrics.sectionLabelSize
                verticalAlignment: Text.AlignVCenter
            }

            Rectangle {
                width: parent.width
                height: bundledSection.implicitHeight
                radius: IOSMetrics.groupedCornerRadius
                color: IOSPalette.elevatedSurface(root.pageBackground)
                clip: true

                Column {
                    id: bundledSection
                    width: parent.width

                    Row {
                        width: parent.width
                        height: IOSMetrics.groupedDetailRowHeight
                        leftPadding: IOSMetrics.tableHorizontalInset
                        rightPadding: IOSMetrics.tableHorizontalInset
                        IOSLabel {
                            width: parent.width - statusLabel.width - 8 - 2 * IOSMetrics.tableHorizontalInset
                            height: parent.height
                            text: "Bundled rootfs"
                            verticalAlignment: Text.AlignVCenter
                        }
                        IOSLabel {
                            id: statusLabel
                            width: Math.min(190, parent.width * 0.50)
                            height: parent.height
                            text: rootfsManager.prepared ? "Ready" : "Preparing"
                            color: rootfsManager.prepared ? IOSPalette.green : IOSPalette.secondaryText(root.pageBackground)
                            horizontalAlignment: Text.AlignRight
                            verticalAlignment: Text.AlignVCenter
                            font.pixelSize: IOSMetrics.rowDetailSize
                        }
                    }

                    Rectangle { width: parent.width; height: 1; color: IOSPalette.separator(root.pageBackground); opacity: 0.55 }

                    IOSButton {
                        width: parent.width
                        height: IOSMetrics.groupedRowHeight
                        text: "Prepare bundled rootfs"
                        enabled: !rootfsManager.prepared
                        background: Rectangle { color: "transparent" }
                        onClicked: rootfsManager.prepare()
                    }

                    Rectangle { width: parent.width; height: 1; color: IOSPalette.separator(root.pageBackground); opacity: 0.55 }

                    IOSButton {
                        width: parent.width
                        height: IOSMetrics.groupedRowHeight
                        text: "Boot bundled rootfs"
                        enabled: rootfsManager.prepared
                        background: Rectangle { color: "transparent" }
                        onClicked: root.bootRootRequested()
                    }
                }
            }

            IOSLabel {
                width: parent.width
                height: IOSMetrics.sectionHeaderHeight
                text: "Imported filesystems"
                color: IOSPalette.secondaryText(root.pageBackground)
                font.pixelSize: IOSMetrics.sectionLabelSize
                verticalAlignment: Text.AlignVCenter
            }

            Rectangle {
                width: parent.width
                height: Math.max(IOSMetrics.groupedRowHeight, rootsView.contentHeight)
                radius: IOSMetrics.groupedCornerRadius
                color: IOSPalette.elevatedSurface(root.pageBackground)
                clip: true

                ListView {
                    id: rootsView
                    anchors.fill: parent
                    model: rootModel
                    clip: true
                    spacing: 0

                    delegate: Item {
                        width: rootsView.width
                        height: IOSMetrics.groupedRowHeight

                        Rectangle {
                            anchors.fill: parent
                            color: rootMouse.containsMouse ? IOSPalette.separator(root.pageBackground) : "transparent"
                        }

                        IOSLabel {
                            anchors.left: parent.left
                            anchors.leftMargin: IOSMetrics.tableHorizontalInset
                            anchors.right: useButton.left
                            anchors.rightMargin: 4
                            anchors.verticalCenter: parent.verticalCenter
                            text: name + (rootModel.defaultRoot === name ? "  (default)" : "")
                            elide: Text.ElideMiddle
                        }

                        IOSToolButton {
                            id: useButton
                            anchors.right: deleteButton.left
                            anchors.verticalCenter: parent.verticalCenter
                            width: 50
                            height: IOSMetrics.minimumTouchTarget
                            text: rootModel.defaultRoot === name ? "✓" : "Use"
                            enabled: rootModel.defaultRoot !== name
                            font.pixelSize: 13
                            onClicked: rootModel.defaultRoot = name
                        }

                        IOSToolButton {
                            id: deleteButton
                            anchors.right: parent.right
                            anchors.rightMargin: 4
                            anchors.verticalCenter: parent.verticalCenter
                            width: 58
                            height: IOSMetrics.minimumTouchTarget
                            text: "Delete"
                            enabled: rootModel.defaultRoot !== name
                            font.pixelSize: 13
                            onClicked: rootModel.destroyRoot(name)
                        }

                        Rectangle {
                            visible: index < rootsView.count - 1
                            anchors.left: parent.left
                            anchors.right: parent.right
                            anchors.bottom: parent.bottom
                            height: 1
                            color: IOSPalette.separator(root.pageBackground)
                            opacity: 0.55
                        }

                        MouseArea {
                            id: rootMouse
                            anchors.fill: parent
                            hoverEnabled: true
                            z: -1
                        }
                    }

                    IOSLabel {
                        anchors.centerIn: parent
                        visible: rootModel.count === 0
                        text: "No additional filesystems"
                        color: IOSPalette.secondaryText(root.pageBackground)
                        font.pixelSize: IOSMetrics.rowDetailSize
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
