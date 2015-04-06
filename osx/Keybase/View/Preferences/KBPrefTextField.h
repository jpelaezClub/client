//
//  KBPrefTextField.h
//  Keybase
//
//  Created by Gabriel on 4/6/15.
//  Copyright (c) 2015 Gabriel Handford. All rights reserved.
//

#import <Foundation/Foundation.h>

#import "KBPrefView.h"

@interface KBPrefTextField : KBPrefView

- (void)setLabelText:(NSString *)labelText infoText:(NSString *)infoText identifier:(NSString *)identifier;

@end
