import QtQuick
import QtQuick.Layouts
import IshQt

IOSPage {
    id: root

    signal editRequested(string themeName)

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

            IOSButton {
                Layout.fillWidth: true
                text: "Edit selected theme"
                onClicked: root.editRequested(preferences.themeName)
            }

            IOSLabel { text: "Font family" }
            IOSTextField {
                Layout.fillWidth: true
                text: preferences.fontFamily
                onEditingFinished: preferences.fontFamily = text
            }

            IOSLabel { text: "Font size: " + preferences.fontSize }
            IOSSlider {
                Layout.fillWidth: true
                from: 6
                to: 32
                value: preferences.fontSize
                onMoved: preferences.fontSize = Math.round(value)
            }

            IOSLabel { text: "Color scheme" }
            IOSComboBox {
                Layout.fillWidth: true
                model: ["Match system", "Always light", "Always dark"]
                currentIndex: preferences.colorScheme
                onActivated: preferences.colorScheme = currentIndex
            }

            IOSLabel { text: "Cursor style" }
            IOSComboBox {
                Layout.fillWidth: true
                model: ["Block", "Beam", "Underline"]
                currentIndex: preferences.cursorStyle
                onActivated: preferences.cursorStyle = currentIndex
            }

            IOSCheckBox {
                text: "Blink cursor"
                checked: preferences.blinkCursor
                onToggled: preferences.blinkCursor = checked
            }

            IOSCheckBox {
                text: "Disable screen dimming"
                checked: preferences.shouldDisableDimming
                onToggled: preferences.shouldDisableDimming = checked
            }

            IOSCheckBox {
                text: "Hide status bar"
                checked: preferences.hideStatusBar
                onToggled: preferences.hideStatusBar = checked
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
