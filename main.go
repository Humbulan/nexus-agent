package main

import (
    "bytes"
    "encoding/json"
    "flag"
    "fmt"
    "net"
    "net/http"
    "os"
    "sync"
    "time"
)

type Service struct {
    Name     string `json:"name"`
    Port     string `json:"port"`
    Protocol string `json:"protocol"`
}

type SlackPayload struct {
    Text string `json:"text"`
}

var lastState = map[string]bool{}

func check(s Service, wg *sync.WaitGroup, ch chan<- Service) {
    defer wg.Done()
    start := time.Now()
	conn, err := net.DialTimeout(s.Protocol, "127.0.0.1:"+s.Port, 1*time.Second)
	ms := float64(time.Since(start).Milliseconds())
    if err != nil {
        s.Protocol = "down"
        recordCheck(s.Name, false, ms)
		ch <- s
        return
    }
    conn.Close()
    s.Protocol = "up"
    recordCheck(s.Name, true, ms)
	ch <- s
}

func sendSlack(webhook string, message string) {
    payload := SlackPayload{Text: message}
    body, _ := json.Marshal(payload)
    resp, err := http.Post(webhook, "application/json", bytes.NewBuffer(body))
    if err != nil {
        fmt.Println("Slack send failed:", err)
        return
    }
    defer resp.Body.Close()
    fmt.Println("Slack alert sent.")
}

func loadServices(path string) ([]Service, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    var services []Service
    if err := json.Unmarshal(data, &services); err != nil {
        return nil, err
    }
    return services, nil
}

func runOnce(services []Service, webhook string, quiet bool) {
	start := time.Now()
	defer func() { recordRun(float64(time.Since(start).Milliseconds())) }()
    var wg sync.WaitGroup
    ch := make(chan Service, len(services))

    for _, s := range services {
        wg.Add(1)
        go check(s, &wg, ch)
    }

    wg.Wait()
    close(ch)

    online := 0
    offline := 0
    var newlyDown []string
    var newlyUp []string

    for s := range ch {
        isUp := s.Protocol == "up"
        wasUp, existed := lastState[s.Name]

        if isUp {
            online++
        } else {
            offline++
        }

        if existed {
            if wasUp && !isUp {
                newlyDown = append(newlyDown, s.Name+" (port "+s.Port+")")
            } else if !wasUp && isUp {
                newlyUp = append(newlyUp, s.Name+" (port "+s.Port+")")
            }
        } else if !isUp {
            newlyDown = append(newlyDown, s.Name+" (port "+s.Port+")")
        }

        lastState[s.Name] = isUp
    }

    if !quiet {
        for _, s := range services {
            status := "[OFFLINE]"
            if lastState[s.Name] {
                status = "[ONLINE] "
            }
            fmt.Printf("%s Port %s: %s\n", status, s.Port, s.Name)
        }
        fmt.Println("-------------------------------------------------------")
        fmt.Printf("STATUS: %d/%d ports verified\n", online, len(services))
        fmt.Println("-------------------------------------------------------")
    }

    if webhook != "" && (len(newlyDown) > 0 || len(newlyUp) > 0) {
        timestamp := time.Now().Format("2006-01-02 15:04:05")
        text := ":rotating_light: *NEXUS AGENT STATE CHANGE* (" + timestamp + ")\n"
        text += fmt.Sprintf("*Summary:* %d online, %d offline of %d\n", online, offline, len(services))
        if len(newlyDown) > 0 {
            text += "\n:red_circle: *WENT DOWN:*\n"
            for _, line := range newlyDown {
                text += "• " + line + "\n"
            }
        }
        if len(newlyUp) > 0 {
            text += "\n:large_green_circle: *RECOVERED:*\n"
            for _, line := range newlyUp {
                text += "• " + line + "\n"
            }
        }
        sendSlack(webhook, text)
    } else if offline > 0 && webhook == "" && !quiet {
        fmt.Println("(No NEXUS_SLACK_WEBHOOK set - skipping Slack alert)")
    }
}

func main() {
    configPath := flag.String("config", "services.json", "path to services config file")
    watch := flag.Bool("watch", false, "run continuously every 30 seconds")
    interval := flag.Duration("interval", 30*time.Second, "interval between checks in watch mode")
    flag.Parse()
    if checkOnly {
    	os.Exit(runCheckMode(*configPath))
    }
    if metricsAddr != "" {
    	startMetricsServer(metricsAddr)
    }

    services, err := loadServices(*configPath)
    if err != nil {
        fmt.Println("Error loading config:", err)
        return
    }

    webhook := os.Getenv("NEXUS_SLACK_WEBHOOK")

    if !*watch {
        fmt.Println("")
        fmt.Println("NEXUS AGENT - DAWN REPORT")
        fmt.Println("-------------------------------------------------------")
        runOnce(services, webhook, false)
        return
    }

    fmt.Printf("NEXUS AGENT - WATCH MODE (%d services, every %s)\n", len(services), interval.String())
    fmt.Println("Press Ctrl+C to stop.")
    fmt.Println("-------------------------------------------------------")

    for {
        start := time.Now()
        runOnce(services, webhook, true)
        elapsed := time.Since(start)

        online := 0
        for _, up := range lastState {
            if up {
                online++
            }
        }
        fmt.Printf("[%s] %d/%d online (checked in %s)\n",
            time.Now().Format("15:04:05"),
            online, len(services),
            elapsed.Round(time.Millisecond))

        sleep := *interval - elapsed
        if sleep > 0 {
            time.Sleep(sleep)
        }
    }
}
