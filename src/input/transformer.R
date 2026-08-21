library(methods)

Transformer <- function(name, type, parameters, input) {
  self <- list(
    name = name,
    type = type,
    parameters = parameters,
    input = input
  )
  class(self) <- "Transformer"
  self
}

transform <- function(self, input) {
  if (self$type == "uppercase") {
    toupper(as.character(input))
  } else if (self$type == "lowercase") {
    tolower(as.character(input))
  } else if (self$type == "replace" && !is.null(self$parameters$old) && !is.null(self$parameters$new)) {
    old <- as.character(self$parameters$old)
    new <- as.character(self$parameters$new)
    gsub(old, new, as.character(input), fixed = TRUE)
  } else {
    stop(sprintf("Transformation type '%s' is not implemented.", self$type))
  }
}

pipeline <- function(transformers, input) {
  result <- input
  for (t in transformers) {
    result <- transform(t, result)
  }
  result
}
