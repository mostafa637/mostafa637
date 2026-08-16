import QtQuick
import QtQuick.Layouts
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

    ColumnLayout {
        anchors.fill: parent
        anchors.margins: 16
        spacing: 10

        IOSLabel {
            text: "Files in the installed rootfs"
            font.pixelSize: 20
        }

        IOSLabel {
            text: rootfsManager.rootPath
            Layout.fillWidth: true
            wrapMode: Text.Wrap
            opacity: 0.75
        }

        ListView {
            id: filesView
            Layout.fillWidth: true
            Layout.fillHeight: true
            model: rootFilesModel
            clip: true
            spacing: 4

            delegate: Rectangle {
                width: filesView.width
                height: 42
                radius: 6
                color: pageBackground
                border.width: 1
                border.color: pageForeground
                opacity: 0.9

                RowLayout {
                    anchors.fill: parent
                    anchors.margins: 8
                    spacing: 8

                    IOSLabel {
                        text: directory ? "[DIR]" : "[FILE]"
                        Layout.preferredWidth: 52
                        opacity: 0.7
                    }
                    IOSLabel {
                        text: name
                        Layout.fillWidth: true
                        elide: Text.ElideMiddle
                    }
                    IOSLabel {
                        text: directory ? "" : String(size)
                        opacity: 0.7
                    }
                }
            }
        }
    }
}
