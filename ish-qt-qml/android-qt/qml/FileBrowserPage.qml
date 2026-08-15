import QtQuick
import QtQuick.Layouts
import IshQt

IOSPage {
    id: root
    header: IOSToolBar { title: "Root files"; onBackClicked: root.closeRequested() }
    ColumnLayout { anchors.fill: parent; anchors.margins: 16; spacing: 10; IOSLabel { text: "Files in the installed rootfs"; font.pixelSize: 20 }; IOSLabel { text: rootfsManager.rootPath; Layout.fillWidth: true; wrapMode: Text.Wrap }; IOSLabel { text: "File browsing is available through the rootfs model when it is prepared."; Layout.fillWidth: true; wrapMode: Text.Wrap } }
}
