#pragma once

#include "RootModel.h"

class RootFilesModel final : public RootModel
{
    Q_OBJECT
public:
    explicit RootFilesModel(QObject *parent = nullptr);
};
