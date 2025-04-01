## http://data.gdeltproject.org/documentation/GDELT-Event_Codebook-V2.0.pdf

I just want an API

http://data.gdeltproject.org/gdeltv2/lastupdate.txt
curl http://data.gdeltproject.org/gdeltv2/lastupdate.txt


curl -s http://data.gdeltproject.org/gdeltv2/lastupdate.txt | head -n 1 | cut -d' ' -f3 | xargs curl -s | funzip
curl http://data.gdeltproject.org/gdeltv2/lastupdate.txt | head -n 2 | cut -d' ' -f3 | xargs curl | funzip
curl http://data.gdeltproject.org/gdeltv2/lastupdate.txt | head -n 3 | cut -d' ' -f3 | xargs curl | funzip

Ok, then find out what the 60 fields mean

https://analysis.gdeltproject.org/module-event-exporter.html
But it's freaking manual!!

tab-delimited lookup files are available that contain the human-friendly textual labels for each of those codes to make it easier to work with the data for those who have not previously worked with CAMEO.

curl http://data.gdeltproject.org/gdeltv2/20240328160000.translation.gkg.csv.zip | funzip | head -n 1

https://www.gdeltproject.org/data.html#rawdatafiles
http://data.gdeltproject.org/documentation/GDELT-Event_Codebook-V2.0.pdf

EventCode. Classify event codes by importance?

QuadClass. Verbal conflict, material conflict.

GoldsteinScale. (floating point) Each CAMEO event code is assigned a numeric score from -10 to +10, capturing the theoretical potential impact that type of event will have on the stability of a country. This is known as the Goldstein Scale. This field specifies the Goldstein score for each event type. NOTE: this score is based on the type of event, not the specifics of the actual event record being recorded – thus two riots, one with 10 people and one with 10,000, will both receive the same Goldstein score. This can be aggregated to various levels of time resolution to yield an approximation of the stability of a location over time 

AvgTone. (numeric) This is the average “tone” of all documents containing one or more
mentions of this event during the 15 minute update in which it was first seen. The score
ranges from -100 (extremely negative) to +100 (extremely positive). Common values range
between -10 and +10, with 0 indicating neutra


http://data.gdeltproject.org/documentation/GDELT-Global_Knowledge_Graph_Codebook-V2.1.pdf
Under GKG 1.0 and 2.0, an article was required to have at least one successfully
identified and geocoded geographic location before it would be included in the GKG output. However,
many topics monitored by GDELT, such as cybersecurity, constitutional discourse, and major policy
discussions, often do not have strong geographic centering, with many articles not mentioning even a
single location. This was excluding a considerable amount of content from the GKG system that is of
high relevance to many GDELT user communities. Thus, beginning with GKG 2.1, an article is included in
the GKG stream if it includes ANY successfully extracted information, INCLUDING GCAM emotional
scores.

Ok, this is very particular.
- Write a driver in Go is the work of a weekend

## http://data.gdeltproject.org/documentation/GDELT-Event_Codebook-V2.0.pdf

GDELT event fields

1. GlobalEventID.
2. Day. 
3. MonthYear. 
4. Year. 
5. FractionDate. 
6. Actor1Code
7. Actor1Name
8. Actor1CountryCode
9. Actor1KnownGroupCode
10. Actor1EthnicCode
11. Actor1Religion1Code
12. Actor1Religion2Code
13. Actor1Type1Code
14. Actor1Type2Code
15. Actor1Type3Code
16. Actor2Code
17. Actor2Name
18. Actor2CountryCode
19. Actor2KnownGroupCode
20. Actor2EthnicCode
21. Actor2Religion1Code
22. Actor2Religion2Code
23. Actor2Type1Code
24. Actor2Type2Code
25. Actor2Type3Code
26. isRootEvent
27. EventCode
28. EventBaseCode
29. EventRootCode
30. QuadClass
31. GoldsteinScale
32. NumMentions
33. NumSources
34. NumArticles
35. AvgTone
36. Actor1Geo_Type
37. Actor1Geo_Fullname
38. Actor1Geo_CountryCode
39. Actor1Geo_ADM1Code
40. Actor1Geo_ADM2Code
41. Actor1Geo_Lat
42. Actor1Geo_Long
43. Actor1Geo_FeatureID
44. Actor2Geo_Type
45. Actor2Geo_Fullname
46. Actor2Geo_CountryCode
47. Actor2Geo_ADM1Code
48. Actor2Geo_ADM2Code
49. Actor2Geo_Lat
50. Actor2Geo_Long
51. Actor2Geo_FeatureID
52. ActionGeo_Type
53. ActionGeo_Fullname
54. ActionGeo_CountryCode
55. ActionGeo_ADM1Code
56. ActionGeo_ADM2Code
57. ActionGeo_Lat
58. ActionGeo_Long
59. ActionGeo_FeatureID
60. DATEADDED
61. SOURCEURL

---

https://web.archive.org/web/20140418033001/http://predictiveheuristics.com/2013/11/12/prediction-and-good-judgment-can-icews-inform-forecasts/
https://predictiveheuristics.com/
https://en.wikipedia.org/wiki/Integrated_Crisis_Early_Warning_System

---

Yeah, ok, then just use the QuadClass + Goldstein scale to filter for urls
Conceivably, combine this with the mentions database

---

https://dataverse.harvard.edu/dataset.xhtml?persistentId=doi%3A10.7910%2FDVN%2FAJGVIT&version=&q=&fileTypeGroupFacet=&fileAccess=&fileSortField=date&tagPresort=false
https://dataverse.harvard.edu/dataset.xhtml?persistentId=doi:10.7910/DVN/28075
https://dataverse.harvard.edu/dataverse/icews


---

curl -s http://data.gdeltproject.org/gdeltv2/lastupdate.txt | head -n 1 | cut -d' ' -f3 | xargs curl -s | funzip | head -n 2 | cut -d'	' -f61
# curl -s http://data.gdeltproject.org/gdeltv2/lastupdate.txt | head -n 2 | tail -n 1 | cut -d' ' -f3 | xargs curl -s | funzip | head -n 2 | cut -d'	' -f61
# curl -s http://data.gdeltproject.org/gdeltv2/lastupdate.txt | head -n 3 | tail -n 1 | cut -d' ' -f3 | xargs curl -s | funzip | head -n 2 | cut -d'	' -f61

---

## http://data.gdeltproject.org/documentation/GDELT-Global_Knowledge_Graph_Codebook-V2.1.pdf

Maybe I want instead to look at the counts 

curl -s http://data.gdeltproject.org/gdeltv2/lastupdate.txt | head -n 3 | tail -n 1 | cut -d' ' -f3 | xargs curl -s | funzip 

 clear && curl -s http://data.gdeltproject.org/gdeltv2/lastupdate.txt | head -n 3 | tail -n 1 | cut -d' ' -f3 | xargs curl -s | funzip | grep "KILL#" | cut -d'   ' -f6 | sort

 clear && curl -s http://data.gdeltproject.org/gdeltv2/lastupdate.txt | head -n 3 | tail -n 1 | cut -d' ' -f3 | xargs curl -s | funzip | grep "KILL#" | cut -d'   ' -f5,6

## Number of news processed by GDELT every 15 minutes

curl -s http://data.gdeltproject.org/gdeltv2/lastupdate.txt | head -n 3 | tail -n 1 | cut -d' ' -f3 | xargs curl -s | funzip | wc -l

