import QtQuick
import QtQuick.Controls as Controls

Controls.Dialog {
    id: dialog
    property string message: ""
    title: "Error"
    modal: true
    standardButtons: Controls.Dialog.Ok
    function showError(titleText, messageText) { title = titleText; message = messageText; open() }
    contentItem: Text { text: dialog.message; wrapMode: Text.Wrap; padding: 16 }
}
