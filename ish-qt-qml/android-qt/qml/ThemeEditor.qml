import QtQuick
import QtQuick.Layouts
import IshQt

IOSPage {
    id: root

    property string originalName: ""
    property var originalTheme: ({})

    function paletteFromTheme(theme) {
        if (theme.shared)
            return theme.shared
        if (theme.light)
            return theme.light
        return ({})
    }

    function loadTheme() {
        originalTheme = themes.themeForName(originalName)
        const palette = paletteFromTheme(originalTheme)
        nameField.text = originalName
        foregroundField.text = palette.foregroundColor || "#ffffff"
        backgroundField.text = palette.backgroundColor || "#000000"
        cursorField.text = palette.cursorColor || ""
        overridesField.text = (palette.colorPaletteOverrides || []).join(", ")
    }

    function makePalette() {
        const palette = {
            foregroundColor: foregroundField.text.trim(),
            backgroundColor: backgroundField.text.trim()
        }
        if (cursorField.text.trim().length > 0)
            palette.cursorColor = cursorField.text.trim()
        const overrideText = overridesField.text.trim()
        if (overrideText.length > 0)
            palette.colorPaletteOverrides = overrideText.split(",").map(function(value) { return value.trim() })
        return palette
    }

    function makeTheme() {
        const palette = makePalette()
        if (originalTheme.light && originalTheme.dark) {
            return {
                version: 1,
                light: palette,
                dark: palette,
                appearance: originalTheme.appearance || undefined
            }
        }
        return {
            version: 1,
            shared: palette,
            appearance: originalTheme.appearance || undefined
        }
    }

    function saveTheme() {
        const newName = nameField.text.trim()
        if (!newName.length)
            return
        const saved = originalName.length && themes.isUserTheme(originalName)
                ? themes.replaceUserTheme(originalName, newName, makeTheme())
                : themes.addUserTheme(newName, makeTheme())
        if (saved)
            root.closeRequested()
    }

    function duplicateTheme() {
        if (themes.duplicateUserTheme(originalName))
            root.closeRequested()
    }

    function deleteTheme() {
        if (themes.deleteUserTheme(originalName))
            root.closeRequested()
    }

    Component.onCompleted: loadTheme()

    header: IOSToolBar {
        title: "Edit theme"
        onBackClicked: root.closeRequested()
    }

    Flickable {
        anchors.fill: parent
        contentWidth: width
        contentHeight: editor.implicitHeight + 32
        clip: true

        ColumnLayout {
            id: editor
            width: parent.width
            anchors.margins: 16
            anchors.left: parent.left
            anchors.right: parent.right
            spacing: 10

            IOSLabel {
                text: root.originalName.length ? root.originalName : "New theme"
                font.pixelSize: 22
                font.bold: true
            }

            IOSTextField {
                id: nameField
                Layout.fillWidth: true
                placeholderText: "Theme name"
            }

            IOSTextField {
                id: foregroundField
                Layout.fillWidth: true
                placeholderText: "Foreground color (#RRGGBB)"
            }

            IOSTextField {
                id: backgroundField
                Layout.fillWidth: true
                placeholderText: "Background color (#RRGGBB)"
            }

            IOSTextField {
                id: cursorField
                Layout.fillWidth: true
                placeholderText: "Cursor color (optional)"
            }

            IOSTextArea {
                id: overridesField
                Layout.fillWidth: true
                Layout.preferredHeight: 84
                placeholderText: "16 ANSI colors, comma-separated (optional)"
            }

            IOSButton {
                Layout.fillWidth: true
                text: "Save"
                onClicked: saveTheme()
            }

            IOSButton {
                visible: originalName.length > 0
                Layout.fillWidth: true
                text: "Duplicate"
                onClicked: duplicateTheme()
            }

            IOSButton {
                visible: originalName.length > 0 && themes.isUserTheme(originalName)
                Layout.fillWidth: true
                text: "Delete"
                onClicked: deleteTheme()
            }
        }
    }
}
