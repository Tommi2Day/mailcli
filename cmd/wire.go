// Package cmd commands
package cmd

import (
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// pflag Value.Type() constants used in both addFlagsToConfig and applyChangedConfigFlags.
const (
	pflagTypeBool        = "bool"
	pflagTypeInt         = "int"
	pflagTypeInt64       = "int64"
	pflagTypeStringSlice = "stringSlice"
	pflagTypeStringArray = "stringArray"
)

// init runs after all other cmd package init() calls — this file sorts last
// alphabetically — so sendCmd.Flags() and imapCmd.PersistentFlags() are fully
// populated by the time we read them.
//
// The flags are copied without shorthands to avoid conflicts (both send and
// imap use -S/-u/-p etc.) and without viper.BindPFlags, since applyChangedConfigFlags
// pushes changed values into viper at highest priority instead.
func init() {
	addFlagsToConfig(sendCmd.Flags())
	addFlagsToConfig(imapCmd.PersistentFlags())
}

// addFlagsToConfig copies all flags from src onto configCmd.PersistentFlags()
// so that config show / config save accept the same override flags as the
// originating commands, without shorthand letters and without viper binding.
func addFlagsToConfig(src *pflag.FlagSet) {
	dst := configCmd.PersistentFlags()
	src.VisitAll(func(f *pflag.Flag) {
		if dst.Lookup(f.Name) != nil {
			return
		}
		switch f.Value.Type() {
		case pflagTypeBool:
			v, _ := src.GetBool(f.Name)
			dst.Bool(f.Name, v, f.Usage)
		case pflagTypeInt:
			v, _ := src.GetInt(f.Name)
			dst.Int(f.Name, v, f.Usage)
		case pflagTypeInt64:
			v, _ := src.GetInt64(f.Name)
			dst.Int64(f.Name, v, f.Usage)
		case pflagTypeStringSlice, pflagTypeStringArray:
			dst.StringSlice(f.Name, nil, f.Usage)
		default:
			dst.String(f.Name, f.DefValue, f.Usage)
		}
	})
}

// applyChangedConfigFlags pushes any configCmd persistent flags that were
// explicitly set on the command line into viper as override values (highest
// priority, above config file and env vars). Call this before effectiveSettings().
func applyChangedConfigFlags() {
	configCmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		if !f.Changed {
			return
		}
		switch f.Value.Type() {
		case pflagTypeBool:
			v, _ := configCmd.PersistentFlags().GetBool(f.Name)
			viper.Set(f.Name, v)
		case pflagTypeInt:
			v, _ := configCmd.PersistentFlags().GetInt(f.Name)
			viper.Set(f.Name, v)
		case pflagTypeInt64:
			v, _ := configCmd.PersistentFlags().GetInt64(f.Name)
			viper.Set(f.Name, v)
		case pflagTypeStringSlice, pflagTypeStringArray:
			v, _ := configCmd.PersistentFlags().GetStringSlice(f.Name)
			viper.Set(f.Name, v)
		default:
			v, _ := configCmd.PersistentFlags().GetString(f.Name)
			viper.Set(f.Name, v)
		}
	})
}
