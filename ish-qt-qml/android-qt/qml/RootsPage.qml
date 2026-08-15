import QtQuick
import QtQuick.Layouts
import IshQt

IOSPage {
    id: root
    header: IOSToolBar { title: "Root filesystems"; onBackClicked: root.closeRequested() }
    ColumnLayout { anchors.fill: parent; anchors.margins: 16; spacing: 10; IOSLabel { text: rootfsManager.prepared ? "Rootfs is ready" : "Rootfs is being prepared"; font.pixelSize: 20 }; IOSLabel { text: rootfsManager.rootPath; Layout.fillWidth: true; wrapMode: Text.Wrap }; IOSButton { text: "Prepare rootfs"; enabled: !rootfsManager.prepared; onClicked: rootfsManager.prepare() }; IOSButton { text: "Boot session"; enabled: rootfsManager.prepared; onClicked: root.bootRootRequested() } }
}
