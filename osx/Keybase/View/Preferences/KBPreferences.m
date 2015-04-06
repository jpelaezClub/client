//
//  KBPreferences.m
//  Keybase
//
//  Created by Gabriel on 2/3/15.
//  Copyright (c) 2015 Gabriel Handford. All rights reserved.
//

#import "KBPreferences.h"

#import "KBAppKit.h"
#import <MASPreferences/MASPreferencesWindowController.h>
#import "AppDelegate.h"

#import "KBPrefGeneralView.h"
#import "KBPrefAdvancedView.h"
#import "KBPrefGPGView.h"

@interface KBPreferences ()
@property MASPreferencesWindowController *preferencesWindowController;
@property NSString *configPath;
@property NSMutableDictionary *config;
@end

@interface KBPreferencesViewController : NSViewController <MASPreferencesViewController>
@property (nonatomic) NSImage *toolbarItemImage;
@property (nonatomic) NSString *toolbarItemLabel;
@end

@implementation KBPreferencesViewController
@end

@implementation KBPreferences

- (void)open:(NSString *)configPath sender:(id)sender {
  _configPath = configPath;
  NSData *configData = [[NSData alloc] initWithContentsOfFile:_configPath];
  if (configData) {
    _config = [NSJSONSerialization JSONObjectWithData:configData options:NSJSONReadingMutableContainers error:nil];
  }

  KBPreferencesViewController *generalViewController = [[KBPreferencesViewController alloc] init];
  generalViewController.view = [[KBPrefGeneralView alloc] initWithPreferences:self];
  generalViewController.view.frame = CGRectMake(0, 0, 600, 400);
  generalViewController.identifier = @"General";
  generalViewController.toolbarItemImage = [NSImage imageNamed:NSImageNamePreferencesGeneral];
  generalViewController.toolbarItemLabel = @"General";

  KBPreferencesViewController *gpgViewController = [[KBPreferencesViewController alloc] init];
  gpgViewController.view = [[KBPrefGPGView alloc] initWithPreferences:self];
  gpgViewController.view.frame = CGRectMake(0, 0, 600, 400);
  gpgViewController.identifier = @"GPG";
  gpgViewController.toolbarItemImage = [NSImage imageNamed:@"gpgtools.icns"];
  gpgViewController.toolbarItemLabel = @"GPG";    

  KBPreferencesViewController *advancedViewController = [[KBPreferencesViewController alloc] init];
  advancedViewController.view = [[KBPrefAdvancedView alloc] initWithPreferences:self];
  advancedViewController.view.frame = CGRectMake(0, 0, 600, 400);
  advancedViewController.identifier = @"Advanced";
  advancedViewController.toolbarItemImage = [NSImage imageNamed:NSImageNameAdvanced];
  advancedViewController.toolbarItemLabel = @"Advanced";

  NSArray *controllers = @[generalViewController, gpgViewController, advancedViewController];
  _preferencesWindowController = [[MASPreferencesWindowController alloc] initWithViewControllers:controllers title:@"Preferences"];

  for (NSViewController *viewController in controllers) {
    [(YOView *)[viewController view] layoutView];
  }

  NSWindow *window = _preferencesWindowController.window;
  [[sender window] kb_addChildWindow:window rect:CGRectMake(0, 0, 600, 400) position:KBWindowPositionCenter fixed:NO];
}

- (void)close {
  [_preferencesWindowController close];
}

- (void)saveConfig {
  NSAssert(_config, @"No config");

  // Don't save, we're be updating how we set config
  //NSData *configData = [NSJSONSerialization dataWithJSONObject:_config options:NSJSONWritingPrettyPrinted error:nil];
  //[configData writeToFile:_configPath atomically:YES];
}

- (id)valueForIdentifier:(NSString *)identifier {
  NSAssert(_config, @"No config");

  if ([identifier isEqualTo:@"Preferences.GPGEnabled"]) {
    return @(![_config[@"gpg-disabled"] boolValue]);
  }

  if ([identifier gh_startsWith:@"Preferences."]) {
    return [NSUserDefaults.standardUserDefaults objectForKey:identifier];
  } else {
    return _config[identifier];
  }
}

- (void)setValue:(id)value forIdentifier:(NSString *)identifier synchronize:(BOOL)synchronize {
  NSAssert(_config, @"No config");

  if ([identifier isEqualTo:@"Preferences.GPGEnabled"]) {
    identifier = @"gpg-disabled";
    value = @(![value boolValue]);
  }

  if ([identifier gh_startsWith:@"Preferences."]) {
    GHDebug(@"Setting (local) %@=%@", identifier, value);
    [NSUserDefaults.standardUserDefaults setObject:value forKey:identifier];
    if (synchronize) [NSUserDefaults.standardUserDefaults synchronize];
  } else {
    GHDebug(@"Setting %@=%@", identifier, value);
    _config[identifier] = value;
    if (synchronize) [self saveConfig];
  }
}

@end
