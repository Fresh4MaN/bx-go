package repo

import sq "github.com/Masterminds/squirrel"

// ps — общий SQL builder с плейсхолдерами MySQL (?).
var ps = sq.StatementBuilder.PlaceholderFormat(sq.Question)
