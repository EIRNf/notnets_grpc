
go mod edit -module notnets_grpc
# -- rename all imported module
find . -type f -name '*.go' \
-exec sed -i '' -e 's/grpcintersectiontest/notnets_grpc/g' {} \;