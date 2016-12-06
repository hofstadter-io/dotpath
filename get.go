package dotpath

import (
	"fmt"
	"github.com/pkg/errors"
	"reflect"
	"strings"

	log "gopkg.in/inconshreveable/log15.v2" // logging framework
)

var logger = log.New()

func init() {
	config_logger("warn")
}

func SetLogLevel(level string) {
	config_logger(level)
}

func config_logger(level string) {
	logger = log.New()
	term_level, err := log.LvlFromString(level)
	if err != nil {
		panic(err)
	}

	/*
		term_stack := log.CallerStackHandler("%+v", log.StdoutHandler)
		term_caller := log.CallerFuncHandler(log.CallerFileHandler(term_stack))
		termlog := log.LvlFilterHandler(term_level, term_caller)
	*/

	term_caller := log.CallerFuncHandler(log.CallerFileHandler(log.StdoutHandler))
	termlog := log.LvlFilterHandler(term_level, term_caller)

	//	termlog := log.LvlFilterHandler(term_level, log.StdoutHandler)
	logger.SetHandler(termlog)

}

func Get(path string, data interface{}) (interface{}, error) {
	paths := strings.Split(path, ".")
	if len(paths) < 1 {
		return nil, errors.New("Bad path supplied: " + path)
	}

	// fmt.Println("GETPATH:", path, paths, data)

	return get_by_path(0, paths, data)
}

func GetByPathSlice(path []string, data interface{}) (interface{}, error) {
	return get_by_path(0, path, data)
}

func get_by_path(IDX int, paths []string, data interface{}) (interface{}, error) {
	header := fmt.Sprintf("get_by_path:  %d  %v  in:\n%+v\n\n", IDX, paths, data)
	// fmt.Println(header)
	logger.Info(header)

	P := paths[IDX]
	path_str := strings.Join(paths[:IDX+1], ".")

	has_indexing := strings.Contains(P, "[")
	has_slicing := strings.Contains(P, ":")
	has_listing := strings.Contains(P, ",")
	has_regex := strings.Contains(P, "regex")

	has_eq := strings.Contains(P, "==")
	has_ne := strings.Contains(P, "!=")
	has_ge := strings.Contains(P, ">=")
	has_gt := strings.Contains(P, ">")
	has_le := strings.Contains(P, "<=")
	has_lt := strings.Contains(P, "<")

	if has_indexing || has_slicing || has_listing || has_regex {
		logger.Info("Has: ", "idx", IDX, "curr", P, "paths", paths,
			"indexing", has_indexing, "slicing", has_regex,
			"listing", has_listing, "select", has_regex,
		)
	}

	if has_eq || has_ne || has_ge || has_gt || has_le || has_lt {
		logger.Info("has bool",
			"has_eq", has_eq, "has_ne", has_ne,
			"has_ge", has_ge, "has_gt", has_gt,
			"has_le", has_le, "has_lt", has_lt,
		)
	}

	switch T := data.(type) {

	case map[string]interface{}:
		val, ok := T[P]
		if !ok {
			// try to look up by name
			name_value, ok := T["name"]
			if ok && name_value == P {
				return T, nil
			}
			return nil, errors.New("could not find '" + P + "' in object")
		}
		add_parent_and_path(val, T, path_str)
		if len(paths) == IDX+1 {
			return val, nil
		}
		ret, err := get_by_path(IDX+1, paths, val)
		if err != nil {
			return nil, errors.Wrapf(err, "from object "+P)
		}
		return ret, nil

	case map[interface{}]interface{}:
		val, ok := T[P]
		if !ok {
			// try to look up by name
			name_value, ok := T["name"]
			if ok && name_value == P {
				return T, nil
			}
			return nil, errors.New("could not find '" + P + "' in object")
		}
		add_parent_and_path(val, T, path_str)
		if len(paths) == IDX+1 {
			return val, nil
		}
		ret, err := get_by_path(IDX+1, paths, val)
		if err != nil {
			return nil, errors.Wrapf(err, "from object "+P)
		}
		return ret, nil

	case []interface{}:
		logger.Info("Processing Slice", "paths", paths, "T", T)
		subs := []interface{}{}
		if len(paths) == IDX+1 {
			for _, elem := range T {
				logger.Info("    - elem", "elem", elem, "paths", paths, "P", P, "elem", elem)
				switch V := elem.(type) {

				case map[string]interface{}:
					logger.Debug("        map[string]")
					val, ok := V[P]
					if !ok {
						// try to look up by name
						name_value, ok := V["name"]
						if ok && name_value == P {
							return T, nil
						}
						logger.Debug("could not find '" + P + "' in object")
						continue
					}

					// accumulate based on type (slice or not)
					switch a_val := val.(type) {

					case []interface{}:
						logger.Debug("Adding vals", "val", a_val)
						subs = append(subs, a_val...)

					default:
						logger.Debug("Adding val", "val", a_val)
						subs = append(subs, a_val)
					}

				case map[interface{}]interface{}:
					logger.Debug("        map[iface]", "P", P, "V", V, "paths", paths)
					val, ok := V[P]
					if !ok {
						// try to look up by name
						name_value, ok := V["name"]
						if ok && name_value == P {
							return T, nil
						}
						logger.Debug("could not find '" + P + "' in object")
						continue
					}

					// accumulate based on type (slice or not)
					switch a_val := val.(type) {

					case []interface{}:
						logger.Debug("Adding vals", "val", a_val)
						subs = append(subs, a_val...)

					default:
						logger.Debug("Adding val", "val", a_val)
						subs = append(subs, a_val)
					}

				default:
					str := fmt.Sprintf("%+v", reflect.TypeOf(V))
					return nil, errors.New("element not an object type: " + str)

				}
			}
		} else {
			for _, elem := range T {
				val, err := get_by_path(IDX, paths, elem)
				if err != nil {
					// in this case, only some of the sub.paths.elements may be found
					// this err path should be expanded to check for geb error types
					logger.Debug(err.Error())
					continue
				}
				switch V := val.(type) {

				case []interface{}:
					logger.Debug("Adding vals", "val", V)
					subs = append(subs, V...)

				default:
					logger.Debug("Adding val", "val", V)
					subs = append(subs, V)

				}
			}
		}

		return subs, nil

	default:
		str := fmt.Sprintf("%+v", reflect.TypeOf(data))
		return nil, errors.New("unknown data object type: " + str)

	} // END of type switch

}

func add_parent_and_path(child interface{}, parent interface{}, path string) (interface{}, error) {
	logger.Info("adding parent to child", "child", child, "parent", parent, "path", path)
	parent_ref := "unknown-parent"
	switch P := parent.(type) {

	case map[string]interface{}:
		p_ref, ok := P["name"]
		if !ok {
			return nil, errors.Errorf("parent does not have name: %+v", parent)
		}
		parent_ref = p_ref.(string)
	case map[interface{}]interface{}:
		p_ref, ok := P["name"]
		if !ok {
			return nil, errors.Errorf("parent does not have name: %+v", parent)
		}
		parent_ref = p_ref.(string)

	default:
		str := fmt.Sprintf("%+v", reflect.TypeOf(parent))
		return nil, errors.New("unknown parent object type: " + str)

	}

	switch C := child.(type) {

	case map[string]interface{}:
		C["parent"] = parent_ref
		C["path"] = path
	case map[interface{}]interface{}:
		C["parent"] = parent_ref
		C["path"] = path

	case []interface{}:
		for _, elem := range C {
			switch E := elem.(type) {
			case map[string]interface{}:
				E["parent"] = parent_ref
				E["path"] = path
			case map[interface{}]interface{}:
				E["parent"] = parent_ref
				E["path"] = path
			default:
				str := fmt.Sprintf("in slice of %+v", reflect.TypeOf(E))
				return nil, errors.New("element not an object type: " + str)
			}
		}

	default:
		str := fmt.Sprintf("%+v", reflect.TypeOf(C))
		return nil, errors.New("unknown data object type: " + str)

	}
	return child, nil
}
