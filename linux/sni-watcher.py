#!/usr/bin/env python3
"""Minimal org.kde.StatusNotifierWatcher so Qt's QSystemTrayIcon.isSystemTrayAvailable() is true in the container.
A stock GNOME session has NO such service (needs the AppIndicator extension); KDE has one. Run it inside a session
(DBUS_SESSION_BUS_ADDRESS set) before starting a Qt app to emulate the KDE case."""
import dbus, dbus.service, dbus.mainloop.glib
from gi.repository import GLib
dbus.mainloop.glib.DBusGMainLoop(set_as_default=True)
IFACE = "org.kde.StatusNotifierWatcher"

class Watcher(dbus.service.Object):
    def __init__(self, bus):
        super().__init__(bus, "/StatusNotifierWatcher")
        self.items = []
    @dbus.service.method(IFACE, in_signature="s")
    def RegisterStatusNotifierItem(self, s):
        self.items.append(s); print("item registered:", s, flush=True)
    @dbus.service.method(IFACE, in_signature="s")
    def RegisterStatusNotifierHost(self, s): pass
    XML = """<node><interface name="org.kde.StatusNotifierWatcher">
<method name="RegisterStatusNotifierItem"><arg type="s" direction="in"/></method>
<method name="RegisterStatusNotifierHost"><arg type="s" direction="in"/></method>
<property name="IsStatusNotifierHostRegistered" type="b" access="read"/>
<property name="ProtocolVersion" type="i" access="read"/>
<property name="RegisteredStatusNotifierItems" type="as" access="read"/>
<signal name="StatusNotifierItemRegistered"><arg type="s"/></signal>
<signal name="StatusNotifierHostRegistered"/>
</interface>
<interface name="org.freedesktop.DBus.Properties">
<method name="Get"><arg type="s" direction="in"/><arg type="s" direction="in"/><arg type="v" direction="out"/></method>
<method name="GetAll"><arg type="s" direction="in"/><arg type="a{sv}" direction="out"/></method>
</interface></node>"""
    @dbus.service.method("org.freedesktop.DBus.Introspectable", out_signature="s")
    def Introspect(self):
        return self.XML
    @dbus.service.method("org.freedesktop.DBus.Properties", in_signature="ss", out_signature="v")
    def Get(self, iface, prop):
        return {"IsStatusNotifierHostRegistered": dbus.Boolean(True), "ProtocolVersion": dbus.Int32(0),
                "RegisteredStatusNotifierItems": dbus.Array(self.items, signature="s")}[prop]
    @dbus.service.method("org.freedesktop.DBus.Properties", in_signature="s", out_signature="a{sv}")
    def GetAll(self, iface):
        return {"IsStatusNotifierHostRegistered": dbus.Boolean(True), "ProtocolVersion": dbus.Int32(0),
                "RegisteredStatusNotifierItems": dbus.Array(self.items, signature="s")}

bus = dbus.SessionBus()
name = dbus.service.BusName(IFACE, bus)
w = Watcher(bus)
print("watcher ready", flush=True)
GLib.MainLoop().run()
