import QtQuick
import QtQuick.Controls as Controls
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
                text: "Terminal appearance"
                color: IOSPalette.secondaryText(root.pageBackground)
                font.pixelSize: IOSMetrics.sectionLabelSize
                font.bold: false
                verticalAlignment: Text.AlignVCenter
            }

            Rectangle {
                width: parent.width
                height: appearanceSection.implicitHeight
                radius: IOSMetrics.groupedCornerRadius
                color: IOSPalette.elevatedSurface(root.pageBackground)
                clip: true

                Column {
                    id: appearanceSection
                    width: parent.width

                    Row {
                        width: parent.width
                        height: IOSMetrics.groupedRowHeight
                        leftPadding: IOSMetrics.tableHorizontalInset
                        rightPadding: IOSMetrics.tableHorizontalInset
                        spacing: 8
                        IOSLabel {
                            width: parent.width - themeCombo.width - 8 - 2 * IOSMetrics.tableHorizontalInset
                            height: parent.height
                            text: "Theme"
                            verticalAlignment: Text.AlignVCenter
                        }
                        IOSComboBox {
                            id: themeCombo
                            width: Math.min(170, parent.width * 0.48)
                            height: parent.height
                            model: themes.themeNames
                            currentIndex: Math.max(0, themes.themeNames.indexOf(preferences.themeName))
                            onActivated: preferences.themeName = currentText
                        }
                    }

                    Rectangle { width: parent.width; height: 1; color: IOSPalette.separator(root.pageBackground); opacity: 0.55 }

                    IOSButton {
                        width: parent.width
                        height: IOSMetrics.groupedRowHeight
                        text: "Edit selected theme"
                        background: Rectangle { color: "transparent" }
                        onClicked: root.editRequested(preferences.themeName)
                    }

                    Rectangle { width: parent.width; height: 1; color: IOSPalette.separator(root.pageBackground); opacity: 0.55 }

                    Row {
                        width: parent.width
                        height: IOSMetrics.groupedRowHeight
                        leftPadding: IOSMetrics.tableHorizontalInset
                        rightPadding: IOSMetrics.tableHorizontalInset
                        spacing: 8
                        IOSLabel {
                            width: parent.width - familyField.width - 8 - 2 * IOSMetrics.tableHorizontalInset
                            height: parent.height
                            text: "Font family"
                            verticalAlignment: Text.AlignVCenter
                        }
                        IOSTextField {
                            id: familyField
                            width: Math.min(190, parent.width * 0.52)
                            height: parent.height - 8
                            anchors.verticalCenter: parent.verticalCenter
                            text: preferences.fontFamily
                            horizontalAlignment: Text.AlignRight
                            onEditingFinished: preferences.fontFamily = text
                        }
                    }

                    Rectangle { width: parent.width; height: 1; color: IOSPalette.separator(root.pageBackground); opacity: 0.55 }

                    Row {
                        width: parent.width
                        height: IOSMetrics.groupedRowHeight
                        leftPadding: IOSMetrics.tableHorizontalInset
                        rightPadding: IOSMetrics.tableHorizontalInset
                        spacing: 8
                        IOSLabel {
                            width: parent.width - fontSlider.width - 8 - 2 * IOSMetrics.tableHorizontalInset
                            height: parent.height
                            text: "Font size  " + preferences.fontSize
                            verticalAlignment: Text.AlignVCenter
                        }
                        IOSSlider {
                            id: fontSlider
                            width: Math.min(150, parent.width * 0.42)
                            anchors.verticalCenter: parent.verticalCenter
                            from: 6
                            to: 32
                            value: preferences.fontSize
                            onMoved: preferences.fontSize = Math.round(value)
                        }
                    }

                    Rectangle { width: parent.width; height: 1; color: IOSPalette.separator(root.pageBackground); opacity: 0.55 }

                    Row {
                        width: parent.width
                        height: IOSMetrics.groupedRowHeight
                        leftPadding: IOSMetrics.tableHorizontalInset
                        rightPadding: IOSMetrics.tableHorizontalInset
                        spacing: 8
                        IOSLabel {
                            width: parent.width - colorCombo.width - 8 - 2 * IOSMetrics.tableHorizontalInset
                            height: parent.height
                            text: "Color scheme"
                            verticalAlignment: Text.AlignVCenter
                        }
                        IOSComboBox {
                            id: colorCombo
                            width: Math.min(170, parent.width * 0.48)
                            height: parent.height
                            model: ["Match system", "Always light", "Always dark"]
                            currentIndex: preferences.colorScheme
                            onActivated: preferences.colorScheme = currentIndex
                        }
                    }

                    Rectangle { width: parent.width; height: 1; color: IOSPalette.separator(root.pageBackground); opacity: 0.55 }

                    Row {
                        width: parent.width
                        height: IOSMetrics.groupedRowHeight
                        leftPadding: IOSMetrics.tableHorizontalInset
                        rightPadding: IOSMetrics.tableHorizontalInset
                        spacing: 8
                        IOSLabel {
                            width: parent.width - cursorCombo.width - 8 - 2 * IOSMetrics.tableHorizontalInset
                            height: parent.height
                            text: "Cursor style"
                            verticalAlignment: Text.AlignVCenter
                        }
                        IOSComboBox {
                            id: cursorCombo
                            width: Math.min(170, parent.width * 0.48)
                            height: parent.height
                            model: ["Block", "Beam", "Underline"]
                            currentIndex: preferences.cursorStyle
                            onActivated: preferences.cursorStyle = currentIndex
                        }
                    }

                    Rectangle { width: parent.width; height: 1; color: IOSPalette.separator(root.pageBackground); opacity: 0.55 }

                    IOSCheckBox {
                        width: parent.width
                        height: IOSMetrics.groupedRowHeight
                        text: "Blink cursor"
                        checked: preferences.blinkCursor
                        onToggled: preferences.blinkCursor = checked
                    }

                    Rectangle { width: parent.width; height: 1; color: IOSPalette.separator(root.pageBackground); opacity: 0.55 }

                    IOSCheckBox {
                        width: parent.width
                        height: IOSMetrics.groupedRowHeight
                        text: "Disable screen dimming"
                        checked: preferences.shouldDisableDimming
                        onToggled: preferences.shouldDisableDimming = checked
                    }

                    Rectangle { width: parent.width; height: 1; color: IOSPalette.separator(root.pageBackground); opacity: 0.55 }

                    IOSCheckBox {
                        width: parent.width
                        height: IOSMetrics.groupedRowHeight
                        text: "Hide status bar"
                        checked: preferences.hideStatusBar
                        onToggled: preferences.hideStatusBar = checked
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
