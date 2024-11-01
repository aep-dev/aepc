terraform {
  required_providers {
    bookstore = {
      source  = "aep.dev/examples/bookstore"
      version = "~> 0.0.1"
    }
  }
}

# resource "bookstore_books" "book" {
#  isbn = ["978-3-16-148410-0"]
#  price = 10.00
#  published = false
# }
