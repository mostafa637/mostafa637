import QtQuick
import QtQuick.Layouts
import IshQt

IOSPage {
    id: root
    property string originalName: ""
    function loadTheme() { }
    header: IOSToolBar { title: "Edit theme"; onBackClicked: root.closeRequested() }
    ColumnLayout { anchors.fill: parent; anchors.margins: 16; spacing: 10; IOSLabel { text: root.originalName.length ? root.originalName : "Theme"; font.pixelSize: 22; font.bold: true }; IOSTextField { Layout.fillWidth: true; placeholderText: "Theme name" }; IOSButton { text: "Save"; onClicked: root.closeRequested() } }
}
