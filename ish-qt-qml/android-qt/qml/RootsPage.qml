import QtQuick
import QtQuick.Layouts
import IshQt

IOSPage {
    id: root

    header: IOSToolBar {
        title: "Root filesystems"
        onBackClicked: root.closeRequested()
    }

    ColumnLayout {
        anchors.fill: parent
        anchors.margins: 16
        spacing: 10

        IOSLabel {
            text: rootfsManager.prepared ? "Bundled rootfs is ready" : "Bundled rootfs is being prepared"
            font.pixelSize: 20
        }

        IOSLabel {
            text: rootfsManager.rootPath
            Layout.fillWidth: true
            wrapMode: Text.Wrap
            opacity: 0.75
        }

        IOSButton {
            text: "Prepare bundled rootfs"
            enabled: !rootfsManager.prepared
            onClicked: rootfsManager.prepare()
        }

        IOSButton {
            text: "Boot bundled rootfs"
            enabled: rootfsManager.prepared
            onClicked: root.bootRootRequested()
        }

        IOSLabel {
            text: "Imported filesystems"
            font.pixelSize: 18
            Layout.topMargin: 8
        }

        IOSLabel {
            visible: rootModel.count === 0
            text: "No additional filesystems have been imported."
            opacity: 0.7
        }

        ListView {
            id: rootsView
            Layout.fillWidth: true
            Layout.fillHeight: true
            model: rootModel
            clip: true
            spacing: 6

            delegate: Rectangle {
                width: rootsView.width
                height: 48
                radius: 8
                color: pageBackground
                border.width: 1
                border.color: pageForeground
                opacity: 0.9

                RowLayout {
                    anchors.fill: parent
                    anchors.margins: 8
                    spacing: 8

                    IOSLabel {
                        Layout.fillWidth: true
                        text: name + (rootModel.defaultRoot === name ? " (default)" : "")
                        elide: Text.ElideMiddle
                    }

                    IOSButton {
                        text: rootModel.defaultRoot === name ? "Default" : "Use"
                        enabled: rootModel.defaultRoot !== name
                        onClicked: rootModel.defaultRoot = name
                    }

                    IOSButton {
                        text: "Delete"
                        enabled: rootModel.defaultRoot !== name
                        onClicked: rootModel.destroyRoot(name)
                    }
                }
            }
        }
    }
}
