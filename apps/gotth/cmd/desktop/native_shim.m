//go:build darwin && native

#import <Cocoa/Cocoa.h>
#import <UserNotifications/UserNotifications.h>
#import <UniformTypeIdentifiers/UniformTypeIdentifiers.h>

// NSUserNotification is used only as the unbundled fallback (see gguixNotify).
#pragma clang diagnostic ignored "-Wdeprecated-declarations"

extern void gguixGoMenuAction(char *name);

static NSWindow *gWindow;
static NSMutableDictionary<NSString *, NSMenuItem *> *gItems;

// ---------------------------------------------------------------- menu

@interface GGUIXMenuTarget : NSObject
- (void)fire:(NSMenuItem *)item;
@end

@implementation GGUIXMenuTarget
- (void)fire:(NSMenuItem *)item {
	gguixGoMenuAction((char *)[(NSString *)item.representedObject UTF8String]);
}
@end

static GGUIXMenuTarget *gTarget;

static NSMenuItem *addItem(NSMenu *menu, NSString *title, NSString *action, NSString *key, NSEventModifierFlags mods) {
	NSMenuItem *it = [[NSMenuItem alloc] initWithTitle:title action:@selector(fire:) keyEquivalent:key];
	it.target = gTarget;
	it.representedObject = action;
	it.keyEquivalentModifierMask = mods;
	[menu addItem:it];
	gItems[action] = it;
	return it;
}

// Standard edit commands go to the first responder (the WKWebView), which is
// how a native text field gets Cut/Copy/Paste/Select All.
static void addStd(NSMenu *menu, NSString *title, SEL sel, NSString *key) {
	[menu addItem:[[NSMenuItem alloc] initWithTitle:title action:sel keyEquivalent:key]];
}

static NSMenu *addMenu(NSMenu *bar, NSString *title) {
	NSMenuItem *top = [[NSMenuItem alloc] initWithTitle:title action:nil keyEquivalent:@""];
	NSMenu *m = [[NSMenu alloc] initWithTitle:title];
	m.autoenablesItems = NO;
	top.submenu = m;
	[bar addItem:top];
	return m;
}

void gguixInstallMenu(void *window) {
	void (^build)(void) = ^{
		gWindow = (__bridge NSWindow *)window;
		gTarget = [GGUIXMenuTarget new];
		gItems = [NSMutableDictionary new];
		NSEventModifierFlags cmd = NSEventModifierFlagCommand;
		NSMenu *bar = [[NSMenu alloc] initWithTitle:@""];
		bar.autoenablesItems = NO;

		NSMenu *app = addMenu(bar, @"GGUIX");
		addItem(app, @"About GGUIX", @"about", @"", 0);
		[app addItem:[NSMenuItem separatorItem]];
		addItem(app, @"Quit GGUIX", @"quit", @"q", cmd);

		NSMenu *file = addMenu(bar, @"File");
		addItem(file, @"Export CSV…", @"export", @"e", cmd);
		addItem(file, @"Import Notes…", @"import-notes", @"o", cmd);
		addItem(file, @"Reveal Last Export", @"reveal-export", @"", 0).enabled = NO;

		NSMenu *edit = addMenu(bar, @"Edit");
		addStd(edit, @"Cut", @selector(cut:), @"x");
		addStd(edit, @"Copy", @selector(copy:), @"c");
		addStd(edit, @"Paste", @selector(paste:), @"v");
		addStd(edit, @"Select All", @selector(selectAll:), @"a");
		[edit addItem:[NSMenuItem separatorItem]];
		addItem(edit, @"Copy Row", @"copy-row", @"c", cmd | NSEventModifierFlagShift);

		NSMenu *view = addMenu(bar, @"View");
		addItem(view, @"Table", @"view-table", @"1", cmd).state = NSControlStateValueOn;
		addItem(view, @"Cards", @"view-cards", @"2", cmd);
		[view addItem:[NSMenuItem separatorItem]];
		NSMenuItem *themeItem = [[NSMenuItem alloc] initWithTitle:@"Theme" action:nil keyEquivalent:@""];
		NSMenu *theme = [[NSMenu alloc] initWithTitle:@"Theme"];
		theme.autoenablesItems = NO;
		themeItem.submenu = theme;
		[view addItem:themeItem];
		addItem(theme, @"System", @"theme-system", @"", 0).state = NSControlStateValueOn;
		addItem(theme, @"Light", @"theme-light", @"", 0);
		addItem(theme, @"Dark", @"theme-dark", @"", 0);
		[view addItem:[NSMenuItem separatorItem]];
		addItem(view, @"Reload", @"reload", @"r", cmd);
		addItem(view, @"Focus Search", @"focus-search", @"f", cmd);

		NSMenu *help = addMenu(bar, @"Help");
		addItem(help, @"Host Info…", @"host", @"i", cmd);

		[NSApp setMainMenu:bar];
	};
	if ([NSThread isMainThread]) build(); else dispatch_sync(dispatch_get_main_queue(), build);
}

void gguixMenuState(const char *view, const char *theme, int exporting, int hasExport) {
	NSString *v = [NSString stringWithUTF8String:view], *t = [NSString stringWithUTF8String:theme];
	dispatch_async(dispatch_get_main_queue(), ^{
		for (NSString *n in @[@"table", @"cards"])
			gItems[[@"view-" stringByAppendingString:n]].state = [n isEqualToString:v] ? NSControlStateValueOn : NSControlStateValueOff;
		for (NSString *n in @[@"system", @"light", @"dark"])
			gItems[[@"theme-" stringByAppendingString:n]].state = [n isEqualToString:t] ? NSControlStateValueOn : NSControlStateValueOff;
		gItems[@"export"].enabled = !exporting;
		gItems[@"reveal-export"].enabled = hasExport != 0;
	});
}

// ---------------------------------------------------------------- dialogs

// Runs a panel as a sheet on the webview's window and blocks the calling
// (server) goroutine until it ends, so the page stays responsive. Returns a
// malloc'd path, or NULL when cancelled.
static char *runPanel(NSSavePanel * (^make)(void)) {
	__block NSString *result = nil;
	dispatch_semaphore_t sem = dispatch_semaphore_create(0);
	dispatch_async(dispatch_get_main_queue(), ^{
		NSSavePanel *p = make();
		void (^done)(NSModalResponse) = ^(NSModalResponse r) {
			if (r == NSModalResponseOK && p.URL) result = p.URL.path;
			dispatch_semaphore_signal(sem);
		};
		if (gWindow) [p beginSheetModalForWindow:gWindow completionHandler:done];
		else done([p runModal]);
	});
	dispatch_semaphore_wait(sem, DISPATCH_TIME_FOREVER);
	return result ? strdup([result UTF8String]) : NULL;
}

char *gguixSaveDialog(const char *defaultName) {
	NSString *name = [NSString stringWithUTF8String:defaultName];
	return runPanel(^NSSavePanel *{
		NSSavePanel *p = [NSSavePanel savePanel];
		p.title = @"Export CSV";
		p.nameFieldStringValue = name;
		p.canCreateDirectories = YES;
		UTType *csv = [UTType typeWithFilenameExtension:@"csv"];
		if (csv) p.allowedContentTypes = @[csv];
		return p;
	});
}

char *gguixOpenDialog(const char *ext) {
	NSString *e = [NSString stringWithUTF8String:ext];
	return runPanel(^NSSavePanel *{
		NSOpenPanel *p = [NSOpenPanel openPanel];
		p.title = @"Import Notes";
		p.canChooseFiles = YES;
		p.canChooseDirectories = NO;
		p.allowsMultipleSelection = NO;
		UTType *t = [UTType typeWithFilenameExtension:e];
		if (t) p.allowedContentTypes = @[t];
		return p;
	});
}

// ---------------------------------------------------------------- clipboard

void gguixCopyText(const char *text) {
	NSString *s = [NSString stringWithUTF8String:text];
	dispatch_sync(dispatch_get_main_queue(), ^{
		NSPasteboard *pb = [NSPasteboard generalPasteboard];
		[pb clearContents];
		[pb setString:s forType:NSPasteboardTypeString];
	});
}

// ---------------------------------------------------------------- notification

// Shows notifications while the app is frontmost (the export finishes with the
// window in front, and the system would otherwise suppress the banner).
@interface GGUIXNoteDelegate : NSObject <UNUserNotificationCenterDelegate, NSUserNotificationCenterDelegate>
@end
@implementation GGUIXNoteDelegate
- (void)userNotificationCenter:(UNUserNotificationCenter *)c willPresentNotification:(UNNotification *)n
         withCompletionHandler:(void (^)(UNNotificationPresentationOptions))h {
	h(UNNotificationPresentationOptionBanner | UNNotificationPresentationOptionList);
}
- (BOOL)userNotificationCenter:(NSUserNotificationCenter *)c shouldPresentNotification:(NSUserNotification *)n {
	return YES;
}
@end

static GGUIXNoteDelegate *gNoteDelegate;

// UNUserNotificationCenter needs an app bundle with an identifier (it raises
// an exception for a bare binary), so `make package` output uses it and an
// unbundled `make run` binary falls back to the deprecated
// NSUserNotificationCenter, which still accepts a bare process.
int gguixNotify(const char *title, const char *body) {
	NSString *t = [NSString stringWithUTF8String:title], *b = [NSString stringWithUTF8String:body];
	if ([NSBundle mainBundle].bundleIdentifier.length > 0) {
		UNUserNotificationCenter *c = [UNUserNotificationCenter currentNotificationCenter];
		static dispatch_once_t once;
		dispatch_once(&once, ^{ gNoteDelegate = [GGUIXNoteDelegate new]; c.delegate = gNoteDelegate; });
		[c requestAuthorizationWithOptions:(UNAuthorizationOptionAlert | UNAuthorizationOptionSound)
		                 completionHandler:^(BOOL granted, NSError *err) {
			if (!granted) return;
			UNMutableNotificationContent *content = [UNMutableNotificationContent new];
			content.title = t;
			content.body = b;
			UNNotificationRequest *req = [UNNotificationRequest requestWithIdentifier:[[NSUUID UUID] UUIDString] content:content trigger:nil];
			[c addNotificationRequest:req withCompletionHandler:nil];
		}];
		return 0;
	}
#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wdeprecated-declarations"
	NSUserNotificationCenter *c = [NSUserNotificationCenter defaultUserNotificationCenter];
	if (!c) return 1;
	static dispatch_once_t once2;
	dispatch_once(&once2, ^{ gNoteDelegate = [GGUIXNoteDelegate new]; c.delegate = gNoteDelegate; });
	NSUserNotification *n = [NSUserNotification new];
	n.title = t;
	n.informativeText = b;
	[c deliverNotification:n];
#pragma clang diagnostic pop
	return 0;
}
