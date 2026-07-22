//go:build linux

package app

/*
#cgo pkg-config: gtk+-3.0 gdk-pixbuf-2.0
#include <gtk/gtk.h>
#include <gdk-pixbuf/gdk-pixbuf.h>
#include <string.h>
#include <stdlib.h>

typedef struct {
	guchar *data;
	gsize len;
} FlashDockIconPayload;

static gboolean flashdock_apply_icon_idle(gpointer user_data) {
	FlashDockIconPayload *p = (FlashDockIconPayload *)user_data;
	if (p == NULL) {
		return G_SOURCE_REMOVE;
	}
	if (p->data != NULL && p->len > 0) {
		GdkPixbufLoader *loader = gdk_pixbuf_loader_new();
		if (loader != NULL) {
			if (gdk_pixbuf_loader_write(loader, p->data, p->len, NULL) &&
			    gdk_pixbuf_loader_close(loader, NULL)) {
				GdkPixbuf *pixbuf = gdk_pixbuf_loader_get_pixbuf(loader);
				if (pixbuf != NULL) {
					gtk_window_set_default_icon(pixbuf);
					GList *list = gtk_window_list_toplevels();
					for (GList *l = list; l != NULL; l = l->next) {
						GtkWindow *win = GTK_WINDOW(l->data);
						if (win != NULL && gtk_widget_get_visible(GTK_WIDGET(win))) {
							gtk_window_set_icon(win, pixbuf);
						}
					}
					g_list_free(list);
				}
			}
			g_object_unref(loader);
		}
	}
	g_free(p->data);
	g_free(p);
	return G_SOURCE_REMOVE;
}

void flashdockSetWindowIconPNG(const void *data, int length) {
	if (data == NULL || length <= 0) {
		return;
	}
	FlashDockIconPayload *p = g_new0(FlashDockIconPayload, 1);
	p->len = (gsize)length;
	p->data = (guchar *)g_malloc((gsize)length);
	memcpy(p->data, data, (size_t)length);
	g_idle_add(flashdock_apply_icon_idle, p);
}
*/
import "C"
import "unsafe"

func setApplicationDockIconPNG(pngBytes []byte) {
	if len(pngBytes) == 0 {
		return
	}
	C.flashdockSetWindowIconPNG(unsafe.Pointer(&pngBytes[0]), C.int(len(pngBytes)))
}
