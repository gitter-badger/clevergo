package clevergo

import (
	"errors"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"reflect"
)

type WebAction struct {
	BaseMiddleware
	app        *Application      // action's application.
	route      string            // action's route.
	methods    []string          // action's allowed methods.
	fullName   string            // action's full name.
	name       string            // action's name.
	prettyName string            // action's pretty name.
	index      int               // action's index of controller methods.
	controller *ControllerInfo   // action's controller.
	handler    httprouter.Handle // action's handle.
}

func NewWebAction(app *Application, route string, methods []string, name string, index int) (*WebAction, error) {
	if ('A' > name[0]) || (name[0] > 'Z') {
		return nil, errors.New("The action's name is invalid: , it's first charater must be a capital letter." + name)
	}

	ai := &WebAction{
		app:        app,
		route:      route,
		methods:    methods,
		fullName:   name,
		index:      index,
		controller: nil,
		handler:    nil,
	}

	ai.name = getActionName(name)
	ai.prettyName = PrettyName(ai.name)

	return ai, nil
}

func (wa *WebAction) Controller() *ControllerInfo {
	return wa.controller
}

func (wa *WebAction) App() *Application {
	return wa.app
}

func (wa *WebAction) PrettyName() string {
	return wa.prettyName
}

func (wa *WebAction) Handle(ctx *Context) {
	// Create controller's reflect value.
	cv := reflect.New(wa.controller.t)

	// Invoke controller's Init() method.
	initMethod := cv.MethodByName("Init")
	initMethod.Call([]reflect.Value{
		reflect.ValueOf(wa),
		reflect.ValueOf(ctx),
	})

	var values []reflect.Value

	// Invoke controller's BeforeAction() method.
	beforeActionMethod := cv.MethodByName("BeforeAction")
	values = beforeActionMethod.Call([]reflect.Value{})
	// The request will be terminated instantly, if BeforeAction() returns false.
	if value, ok := values[0].Interface().(bool); !ok || !value {
		return
	}

	// Invoke controller's action.
	actionMethod := cv.Method(wa.index) // MethodByIndex is faster than MethodByName.
	// actionMethod := cv.MethodByName(a.fullName)
	actionMethod.Call([]reflect.Value{})

	// Invoke controller's BeforeResponse() method.
	beforeResponseMethod := cv.MethodByName("BeforeResponse")
	beforeResponseMethod.Call([]reflect.Value{})

	return
}

func GenerateWebActionHandler(a *WebAction) httprouter.Handle {
	return func(rw http.ResponseWriter, r *http.Request, params httprouter.Params) {
		ctx := NewContext(a.app, rw, r, params)

		defer ctx.Flush()

		if Configuration.enableLog {
			ctx.Log = a.app.logger.NewLog()
			defer ctx.Log.Flush()
		}

		if a.app.firstMiddleware != nil {
			handler := a.app.firstMiddleware
			handler.Final().SetNext(a)
			handler.Handle(ctx)
		} else {
			a.Handle(ctx)
		}
		return
	}
}

type WebActionRoute struct {
	Route   string
	Methods []string
}

func NewWebActionRoute(route string, args ...[]string) WebActionRoute {
	methods := []string{"GET", "POST"}
	if len(args) > 0 {
		methods = args[0]
	}

	return WebActionRoute{
		Route:   route,
		Methods: methods,
	}
}
