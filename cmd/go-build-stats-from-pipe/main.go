package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/fatih/color"
)

type Server struct {
	Location string `json:"location"`
	Mode     string `json:"mode"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Vms      string `json:"vms"`
}

type stat struct {
	Name  string
	Count int
}

func main() {
	if err := run(os.Stdin); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(r io.Reader) error {
	var servers []Server
	if err := json.NewDecoder(r).Decode(&servers); err != nil {
		return fmt.Errorf("failed to decode json: %w", err)
	}

	nonNormalStats, maintenanceStats, maintenanceByTypeStats, nonNormalByModeStats, nonIncomeStats, freezeEnvStats := processServers(servers)
	printStats(nonNormalStats, maintenanceStats, maintenanceByTypeStats, nonNormalByModeStats, nonIncomeStats, freezeEnvStats)

	return nil
}

func processServers(servers []Server) (map[string]int, map[string]int, map[string]map[string]int, map[string]map[string]int, map[string]int, map[string]int) {
	nonNormalStats := make(map[string]int)
	maintenanceStats := make(map[string]int)
	maintenanceByTypeStats := make(map[string]map[string]int)
	nonNormalByModeStats := make(map[string]map[string]int)
	nonIncomeStats := make(map[string]int)
	freezeEnvStats := make(map[string]int)

	for _, server := range servers {
		if server.Mode != "AGENT_MODE_NORMAL" {
			nonNormalStats[server.Location]++
			if nonNormalByModeStats[server.Location] == nil {
				nonNormalByModeStats[server.Location] = make(map[string]int)
			}
			nonNormalByModeStats[server.Location][server.Mode]++
		}

		if server.Mode == "AGENT_MODE_MAINTENANCE" {
			maintenanceStats[server.Location]++
			if maintenanceByTypeStats[server.Location] == nil {
				maintenanceByTypeStats[server.Location] = make(map[string]int)
			}
			maintenanceByTypeStats[server.Location][server.Type]++
		}

		if (server.Vms == "" && server.Mode != "AGENT_MODE_NORMAL") || server.Mode == "AGENT_MODE_MAINTENANCE" || server.Mode == "AGENT_MODE_SETUP" || server.Mode == "AGENT_MODE_NOT_READY" {
			nonIncomeStats[server.Location]++
		}

		if server.Mode == "AGENT_MODE_FREEZE_ENV" && server.Vms == "" {
			freezeEnvStats[server.Location]++
		}
	}
	return nonNormalStats, maintenanceStats, maintenanceByTypeStats, nonNormalByModeStats, nonIncomeStats, freezeEnvStats
}

func printStats(nonNormalStats, maintenanceStats map[string]int, maintenanceByTypeStats map[string]map[string]int, nonNormalByModeStats map[string]map[string]int, nonIncomeStats map[string]int, freezeEnvStats map[string]int) {
	yellow := color.New(color.FgYellow).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	var nonNormalCount int
	for _, count := range nonNormalStats {
		nonNormalCount += count
	}

	fmt.Printf(yellow("Total servers with non-normal mode: %d\n"), nonNormalCount)
	fmt.Println(cyan("------------------------------------"))
	fmt.Println(cyan("Breakdown by location and mode:"))

	sortedLocations := make([]stat, 0, len(nonNormalByModeStats))
	for location, byMode := range nonNormalByModeStats {
		var total int
		total = 0
		for _, count := range byMode {
			total += count
		}
		sortedLocations = append(sortedLocations, stat{Name: location, Count: total})
	}
	sort.Slice(sortedLocations, func(i, j int) bool {
		return sortedLocations[i].Count > sortedLocations[j].Count
	})

	for _, locationStat := range sortedLocations {
		location := locationStat.Name
		byMode := nonNormalByModeStats[location]
		fmt.Printf("- %s:\n", green(location))

		modeStats := make([]stat, 0, len(byMode))
		for mode, count := range byMode {
			modeStats = append(modeStats, stat{Name: mode, Count: count})
		}
		sort.Slice(modeStats, func(i, j int) bool {
			return modeStats[i].Count > modeStats[j].Count
		})

		for _, modeStat := range modeStats {
			fmt.Printf("  - %s: %s\n", cyan(modeStat.Name), red(modeStat.Count))
		}
	}

	var nonIncomeCount int
	for _, count := range nonIncomeStats {
		nonIncomeCount += count
	}

	fmt.Printf(yellow("\nTotal Servers that CANNOT generate Income: %d\n"), nonIncomeCount)
	fmt.Println(cyan("------------------------------------"))
	fmt.Println(cyan("Breakdown by location:"))

	sortedNonIncomeLocations := make([]stat, 0, len(nonIncomeStats))
	for location, count := range nonIncomeStats {
		sortedNonIncomeLocations = append(sortedNonIncomeLocations, stat{Name: location, Count: count})
	}
	sort.Slice(sortedNonIncomeLocations, func(i, j int) bool {
		return sortedNonIncomeLocations[i].Count > sortedNonIncomeLocations[j].Count
	})

	for _, locationStat := range sortedNonIncomeLocations {
		fmt.Printf("- %s: %s\n", green(locationStat.Name), red(locationStat.Count))
	}

	var freezeEnvCount int
	for _, count := range freezeEnvStats {
		freezeEnvCount += count
	}

	fmt.Printf(yellow("\nTotal servers in AGENT_MODE_FREEZE_ENV empty: %d\n"), freezeEnvCount)
	fmt.Println(cyan("------------------------------------ "))
	fmt.Println(cyan("Breakdown by location:"))

	sortedFreezeEnvLocations := make([]stat, 0, len(freezeEnvStats))
	for location, count := range freezeEnvStats {
		sortedFreezeEnvLocations = append(sortedFreezeEnvLocations, stat{Name: location, Count: count})
	}
	sort.Slice(sortedFreezeEnvLocations, func(i, j int) bool {
		return sortedFreezeEnvLocations[i].Count > sortedFreezeEnvLocations[j].Count
	})

	for _, locationStat := range sortedFreezeEnvLocations {
		fmt.Printf("- %s: %s\n", green(locationStat.Name), red(locationStat.Count))
	}

	var maintenanceCount int
	for _, count := range maintenanceStats {
		maintenanceCount += count
	}

	fmt.Printf(yellow("\nTotal servers in AGENT_MODE_MAINTENANCE: %d\n"), maintenanceCount)
	fmt.Println(cyan("------------------------------------"))
	fmt.Println(cyan("Servers in AGENT_MODE_MAINTENANCE per location:"))

	sortedMaintenanceLocations := make([]stat, 0, len(maintenanceStats))
	for location, count := range maintenanceStats {
		sortedMaintenanceLocations = append(sortedMaintenanceLocations, stat{Name: location, Count: count})
	}
	sort.Slice(sortedMaintenanceLocations, func(i, j int) bool {
		return sortedMaintenanceLocations[i].Count > sortedMaintenanceLocations[j].Count
	})

	for _, locationStat := range sortedMaintenanceLocations {
		fmt.Printf("- %s: %s\n", green(locationStat.Name), red(locationStat.Count))
	}

	fmt.Println(cyan("\nServers in AGENT_MODE_MAINTENANCE per location and type:"))
	fmt.Println(cyan("------------------------------------"))

	sortedMaintenanceTypeLocations := make([]stat, 0, len(maintenanceByTypeStats))
	for location, byType := range maintenanceByTypeStats {
		var total int
		for _, count := range byType {
			total += count
		}
		sortedMaintenanceTypeLocations = append(sortedMaintenanceTypeLocations, stat{Name: location, Count: total})
	}
	sort.Slice(sortedMaintenanceTypeLocations, func(i, j int) bool {
		return sortedMaintenanceTypeLocations[i].Count > sortedMaintenanceTypeLocations[j].Count
	})

	for _, locationStat := range sortedMaintenanceTypeLocations {
		location := locationStat.Name
		byType := maintenanceByTypeStats[location]
		fmt.Printf("- %s:\n", green(location))

		typeStats := make([]stat, 0, len(byType))
		for typeName, count := range byType {
			typeStats = append(typeStats, stat{Name: typeName, Count: count})
		}

		sort.Slice(typeStats, func(i, j int) bool {
			return typeStats[i].Count > typeStats[j].Count
		})

		for _, typeStat := range typeStats {
			fmt.Printf("  - %s: %s\n", cyan(typeStat.Name), red(typeStat.Count))
		}
	}
}
