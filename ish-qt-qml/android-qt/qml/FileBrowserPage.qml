import QtQuick
import IshQt

IOSPage {
    id: root

    header: IOSToolBar {
        title: "Root files"
        onBackClicked: root.closeRequested()
    }

    Component.onCompleted: {
        rootFilesModel.rootPath = rootfsManager.rootPath
        rootFilesModel.refresh()
    }

    Connections {
        target: rootfsManager
        function onRootPathChanged() {
            rootFilesModel.rootPath = rootfsManager.rootPath
            rootFilesModel.refresh()
        }
        function onPreparedChanged() {
            if (rootfsManager.prepared)
                rootFilesModel.refresh()
        }
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
                text: "Files in the installed rootfs"
                color: IOSPalette.secondaryText(root.pageBackground)
                font.pixelSize: IOSMetrics.sectionLabelSize
                verticalAlignment: Text.AlignVCenter
            }

            IOSLabel {
                width: parent.width
                text: rootfsManager.rootPath
                color: IOSPalette.secondaryText(root.pageBackground)
                font.pixelSize: IOSMetrics.rowDetailSize
                wrapMode: Text.Wrap
                elide: Text.ElideMiddle
            }

            Rectangle {
                width: parent.width
                height: Math.max(IOSMetrics.groupedRowHeight, filesView.contentHeight)
                radius: IOSMetrics.groupedCornerRadius
                color: IOSPalette.elevatedSurface(root.pageBackground)
                clip: true

                ListView {
                    id: filesView
                    anchors.fill: parent
                    model: rootFilesModel
                    clip: true
                    spacing: 0

                    delegate: Item {
                        width: filesView.width
                        height: IOSMetrics.groupedRowHeight

                        Rectangle {
                            anchors.fill: parent
                            color: fileMouse.containsMouse ? IOSPalette.separator(root.pageBackground) : "transparent"
                        }

                        IOSLabel {
                            anchors.left: parent.left
                            anchors.leftMargin: IOSMetrics.tableHorizontalInset
                            anchors.verticalCenter: parent.verticalCenter
                            width: 52
                            text: directory ? "[DIR]" : "[FILE]"
                            color: IOSPalette.secondaryText(root.pageBackground)
                            font.pixelSize: IOSMetrics.rowDetailSize
                        }

                        IOSLabel {
                            anchors.left: parent.left
                            anchors.leftMargin: IOSMetrics.tableHorizontalInset + 58
                            anchors.right: sizeLabel.left
                            anchors.rightMargin: 8
                            anchors.verticalCenter: parent.verticalCenter
                            text: name
                            elide: Text.ElideMiddle
                        }

                        IOSLabel {
                            id: sizeLabel
                            anchors.right: parent.right
                            anchors.rightMargin: IOSMetrics.tableHorizontalInset
                            anchors.verticalCenter: parent.verticalCenter
                            text: directory ? "" : String(size)
                            color: IOSPalette.secondaryText(root.pageBackground)
                            font.pixelSize: IOSMetrics.rowDetailSize
                            horizontalAlignment: Text.AlignRight
                        }

                        Rectangle {
                            visible: index < filesView.count - 1
                            anchors.left: parent.left
                            anchors.right: parent.right
                            anchors.bottom: parent.bottom
                            height: 1
                            color: IOSPalette.separator(root.pageBackground)
                            opacity: 0.55
                        }

                        MouseArea {
                            id: fileMouse
                            anchors.fill: parent
                            hoverEnabled: true
                        }
                    }

                    IOSLabel {
                        anchors.centerIn: parent
                        visible: rootFilesModel.count === 0
                        text: "No files"
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
