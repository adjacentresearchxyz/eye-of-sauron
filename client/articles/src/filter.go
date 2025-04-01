					filter := a.getInput("Enter filter keyword: ")
					if filter != "" {
						a.statusMessage = "Filtering items..."
						a.draw()

						// Add to filters file
						// f, err := os.OpenFile("src/filters.txt", os.O_APPEND|os.O_WRONLY, 0644)
						// if err == nil {
						// 	_, err = f.WriteString("\n" + filter)
						// 	f.Close()
						// }
						// if err != nil {
					  // 		log.Printf("Error writing filter: %v", err)
						// }

						// Filter items locally and mark them in server
						var remaining_sources []Source
						for _, source := range a.sources {
							if strings.Contains(strings.ToLower(source.Title), strings.ToLower(filter)) {
								go markProcessedInSever(true, source.ID)
							} else {
								remaining_sources = append(remaining_sources, source)
							}
						}
						a.sources = remaining_sources

