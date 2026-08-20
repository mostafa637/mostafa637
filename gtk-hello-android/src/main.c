#include <gtk/gtk.h>

static void
activate(GtkApplication *app, gpointer user_data)
{
    (void) user_data;
    g_message("GTK Hello World: activate started");

    GtkWidget *window = gtk_application_window_new(app);
    gtk_window_set_title(GTK_WINDOW(window), "GTK Hello World");
    gtk_window_set_default_size(GTK_WINDOW(window), 360, 640);

    GtkWidget *label = gtk_label_new("Hello World");
    gtk_widget_set_halign(label, GTK_ALIGN_CENTER);
    gtk_widget_set_valign(label, GTK_ALIGN_CENTER);
    gtk_window_set_child(GTK_WINDOW(window), label);

    gtk_window_present(GTK_WINDOW(window));
    g_message("GTK Hello World: window presented");
}

int
main(int argc, char **argv)
{
    g_message("GTK Hello World: main started");
    GtkApplication *app = gtk_application_new(
        "com.example.GtkHelloWorld",
        G_APPLICATION_DEFAULT_FLAGS
    );

    g_signal_connect(app, "activate", G_CALLBACK(activate), NULL);

    int status = g_application_run(G_APPLICATION(app), argc, argv);
    g_object_unref(app);

    return status;
}
