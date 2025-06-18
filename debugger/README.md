# debugger

Package debugger implements a simple interactive debugger for [starlark-go](https://github.com/google/starlark-go).
It can be used in any starlark-go application, not necessarily one that uses skycfg.

## Example session

```
❯ go test -c github.com/stripe/skycfg/debugger && ./debugger.test -example
[<prog>:6:19] breakpoint: Stopped
[<prog>:6:19] Available functions: c[ont[inue_]] (or Ctrl+D), q[uit], w[here]/b[ack]t[race], d[own], u[p], f[rame], vars, globals, locals
>>> bt()
Traceback (most recent call last):
   1. <prog>:10:4: in <toplevel>
>  0. <prog>:6:19: in f
>>> vars()
All predeclared symbols in frame 0 (f):
  predeclared: string = "value"
  breakpoint: builtin_function_or_method = <built-in function breakpoint>

All globals in frame 0 (f):
  g: function = <function g>

All free vars in frame 0 (f):
  closure (<prog>:3:5): list = [42]

All locals in frame 0 (f):
  local_var (<prog>:5:9): int = 1
>>> closure
[42]
>>> closure.append(local_var)
>>> closure
[42, 1]
```

### Switch to a different frame

```
>>> f(1)
breakpoint: Stopped at <prog>:10:4: in <toplevel>
>>> bt()
Traceback (most recent call last):
>  1. <prog>:10:4: in <toplevel>
   0. <prog>:6:19: in f
>>> vars()
All predeclared symbols in frame 1 (<toplevel>):
  predeclared: string = "value"
  breakpoint: builtin_function_or_method = <built-in function breakpoint>

All globals in frame 1 (<toplevel>):
  g: function = <function g>

All free vars in frame 1 (<toplevel>):

All locals in frame 1 (<toplevel>):
```

### Continue execution
Notice that the change to variable `closure` is persisted.

```
>>> cont()
closure=[42, 1]
```
