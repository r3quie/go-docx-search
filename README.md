# docx-search
Docx-search is an app written in Go made for better and faster full text searching of doc(x) files. In my experience, full text search in *file explorer* tends to be quite buggy, especially when dealing with numbers. So I decided to make my own "search engine".

It was made with precision in mind, I needed to find specific provisions that were used in previously made documents, so the search does **not** support wildcards.

## Implementation
The code gets the text itself from the .docx file via the archive/zip package. A .docx file is just a zip file containing a bunch of .xml files along with other data of the document like images. So it just opens the file as a zip file and looks through it.

After obtaining the .xml file containing the text body, we're using a regular expression to extract the document body from the file without any XML formatting data. It very simply finds any occurrences of < and > (non-greedily) and deletes everything in between including the inequality signs.

After that, it's just basic array work, finding a substring in a string etc.

The user input is split into a []string by line (\n). After that, we walk the directory specified in env/env when it finds another directory, it'll walk it too. If a .docx file is found the code proceeds as specified above.

Additionally, the search supports a boolean filter. At the moment the search checks one bool to determine whether to apply the filter and after that, it checks the second bool and uses it as the filter. This was relevant because under Czech law, there are 2 types of *subjects*. 

## GUI
The GUI is made using Fyne. Every piece of code, however, should work without the main function. You're welcome to implement it however you like.
