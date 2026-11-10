//go:build darwin

#import <Foundation/Foundation.h>

void gguixDisableAppNap(void) {
	[[NSProcessInfo processInfo] beginActivityWithOptions:NSActivityUserInitiatedAllowingIdleSystemSleep
	                                                 reason:@"GGUIX UI timers must keep running"];
}
