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

	lpos_index := strings.Index(P, "[")
	rpos_index := strings.LastIndex(P, "]")
	pos_colon := strings.Index(P, ":")
	has_listing := strings.Contains(P, ",")
	pos_regex := strings.Index(P, "regex")

	has_eq := strings.Contains(P, "==")
	has_ne := strings.Contains(P, "!=")
	has_ge := strings.Contains(P, ">=")
	has_gt := strings.Contains(P, ">")
	has_le := strings.Contains(P, "<=")
	has_lt := strings.Contains(P, "<")

	logger.Info("Has: ", "idx", IDX, "curr", P, "paths", paths,
		"lpos", lpos_index, "rpos", rpos_index, "slicing", pos_colon,
		"listing", has_listing, "regex", pos_regex,
	)
	logger.Info("has bool",
		"has_eq", has_eq, "has_ne", has_ne,
		"has_ge", has_ge, "has_gt", has_gt,
		"has_le", has_le, "has_lt", has_lt,
	)

	inner := ""
	if lpos_index > -1 {
		inner = P[lpos_index+1 : rpos_index]
	}
	fmt.Printf("inner: %d %q %q\n", IDX, inner, P)

	switch T := data.(type) {

	case map[string]interface{}:
		elems, err := get_from_smap_by_path(IDX, paths, T)
		if err != nil {
			return nil, errors.Wrap(err, "while extracting path from smap in get_by_path")
		}
		return elems, nil

	case map[interface{}]interface{}:
		elems, err := get_from_imap_by_path(IDX, paths, T)
		if err != nil {
			return nil, errors.Wrap(err, "while extracting path from imap in get_by_path")
		}
		return elems, nil

	case []interface{}:
		logger.Info("Processing Slice", "paths", paths, "T", T)
		elems, err := get_from_slice_by_path(IDX, paths, T)
		if err != nil {
			return nil, errors.Wrap(err, "while extracting path from slice")
		}
		if len(paths) == IDX+1 {
			return elems, nil
		} else {
			switch E := elems.(type) {
			case []interface{}:
				ees := []interface{}{}
				for _, e := range E {
					ee, eerr := get_by_path(IDX+1, paths, e)
					if eerr == nil {
						ees = append(ees, ee)
					}
				}
				return ees, nil
			default:
				return get_by_path(IDX+1, paths, elems)
			}
		}

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
