# package supervisor

Package `supervisor` coordinates shutdown among multiple services and the OS.

# Example

```
func main() {
	m := supervisor.New(context.Background())

	// setup
	m.Defer(func(ctx context.Context) error { return doWork(ctx, 1000) }, supervisor.QuietExitFunc)
	m.Defer(func(ctx context.Context) error { return faulty() }, supervisor.QuietExitFunc)

	// launch
	m.Join()
}

func doWork(ctx context.Context, n int) error {
    for i := 0; i < n; i++ {
        // cancelled?
        select {
        case <-ctx.Done():
            return nil
        default:
        }

        fmt.Println(i)
        time.Sleep(10 * time.Millisecond)
    }
}

func faulty() error {
    time.Sleep(100 * time.Millisecond)
    panic("too hard")
}
```
