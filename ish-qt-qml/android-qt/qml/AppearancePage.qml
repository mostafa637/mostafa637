import QtQuick
import QtQuick.Layouts
import IshQt

IOSPage {
    id: root

    header: IOSToolBar {
        title: "Appearance"
        onBackClicked: root.closeRequested()
    }

    Flickable {
        id: flick
        anchors.fill: parent
        contentWidth: width
        contentHeight: column.implicitHeight + 32
        clip: true
        boundsBehavior: Flickable.StopAtBounds

        ColumnLayout {
            id: column
            width: flick.width - 32
            x: 16
            y: 16
            spacing: 12

            IOSLabel { text: "Terminal appearance"; font.pixelSize: 22; font.bold: true }
            IOSLabel { text: "Theme" }
            IOSComboBox {
                Layout.fillWidth: true
                model: themes.themeNames
                currentIndex: Math.max(0, themes.themeNames.indexOf(preferences.themeName))
                onActivated: preferences.themeName = currentText
            }
            IOSLabel { text: "Font size" }
            IOSSlider {
                Layout.fillWidth: true
                from: 6
                to: 32
                value: preferences.fontSize
                onMoved: preferences.fontSize = Math.round(value)
            }
            IOSCheckBox {
                text: "Blink cursor"
                checked: preferences.blinkCursor
                onToggled: preferences.blinkCursor = checked
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
